package agentsrv

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"rscc/internal/agentsrv/mux"
	"rscc/internal/agentsrv/mux/tls"
	"rscc/internal/common/constants"
	"rscc/internal/common/logger"
	"strconv"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type AgentMux struct {
	connQueue   chan net.Conn
	connSem     *semaphore.Weighted
	tcpListener *net.TCPListener
	address     string
	dataPath    string
	mux         *mux.Mux
	lg          *zap.SugaredLogger
}

type AgentMuxParams struct {
	Host        string
	Port        int
	DataPath    string
	TlsCertPath string
	TlsKeyPath  string
}

func NewAgentMux(ctx context.Context, params *AgentMuxParams) (*AgentMux, error) {
	lg := logger.FromContext(ctx).Named("agent")

	address := net.JoinHostPort(params.Host, strconv.Itoa(params.Port))

	muxConfig := &mux.MuxConfig{
		TlsConfig: &tls.ProtocolConfig{
			TlsCertPath: params.TlsCertPath,
			TlsKeyPath:  params.TlsKeyPath,
		},
	}
	mux, err := mux.NewMux(lg, muxConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create mux: %w", err)
	}

	return &AgentMux{
		connQueue: make(chan net.Conn),
		connSem:   semaphore.NewWeighted(constants.MaxUnwrapConnections),
		address:   address,
		dataPath:  params.DataPath,
		lg:        lg,
		mux:       mux,
	}, nil
}

func (a *AgentMux) Start(ctx context.Context) error {
	lg := a.lg

	listenerConfig := &net.ListenConfig{
		KeepAlive: time.Duration(constants.SshTimeout) * time.Second,
	}
	listener, err := listenerConfig.Listen(ctx, "tcp", a.address)
	if err != nil {
		return fmt.Errorf("unable to start agent listener on %s: %w", a.address, err)
	}
	a.tcpListener = listener.(*net.TCPListener)
	lg.Infof("Listener started on %s", a.address)

	g, ctx := errgroup.WithContext(ctx)

	// Start mux
	g.Go(func() error { return a.mux.Start(ctx) })

	// Accept connections
	g.Go(func() error { return a.acceptLoop(ctx) })

	// Unwrap connections
	g.Go(func() error { return a.unwrapLoop(ctx) })

	// Stop listener on context cancel
	g.Go(func() error {
		<-ctx.Done()
		if err := a.closeListener(); err != nil {
			lg.Errorf("Failed to close listener: %v", err)
		}
		lg.Info("Stop listener")
		return ctx.Err()
	})

	return g.Wait()
}

func (a *AgentMux) closeListener() error {
	if a.tcpListener != nil {
		return a.tcpListener.Close()
	}
	return nil
}

func (a *AgentMux) acceptLoop(ctx context.Context) error {
	lg := a.lg

	for {
		conn, err := a.tcpListener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return nil
			}
			lg.Errorf("Failed to accept connection: %v", err)
			continue
		}
		lg.Debugf("Accepted connection from %s", conn.RemoteAddr())

		select {
		case a.connQueue <- conn:
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second): // TODO: Longer timeout?
			lg.Warnf("Connection from %s timed out", conn.RemoteAddr())
			conn.Close()
		}
	}
}

func (a *AgentMux) unwrapLoop(ctx context.Context) error {
	lg := a.lg

	for {
		select {
		case conn := <-a.connQueue:
			if conn == nil {
				continue
			}

			if !a.connSem.TryAcquire(1) {
				lg.Warnf("More than %d connections opened. Skipping connection from %s", constants.MaxUnwrapConnections, conn.RemoteAddr())
				conn.Close()
				continue
			}

			go func(conn net.Conn) {
				defer a.connSem.Release(1)

				protocol, conn, err := a.unwrapConnection(conn)
				if err != nil {
					lg.Errorf("Failed to unwrap connection: %v", err)
					conn.Close()
					return
				}

				if err := protocol.Handle(conn); err != nil {
					lg.Errorf("Failed to handle connection: %v", err)
					conn.Close()
					return
				}
			}(conn)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

type peekableConn struct {
	net.Conn
	reader io.Reader
}

func (pc *peekableConn) Read(b []byte) (n int, err error) {
	return pc.reader.Read(b)
}

func (a *AgentMux) unwrapConnection(conn net.Conn) (mux.Protocol, net.Conn, error) {
	lg := a.lg

	for i := 0; i < constants.MaxUnwrapDepth; i++ {
		protocol, reader, err := a.determineProtocol(conn)
		if err != nil {
			return nil, nil, fmt.Errorf("determine protocol: %w", err)
		}

		if protocol.IsUnwrapped() {
			lg.Debugf("Successfully unwrapped %s protocol from %s", protocol.GetName(), conn.RemoteAddr())
			return protocol, conn, nil
		}

		conn, err = protocol.Unwrap(&peekableConn{
			Conn:   conn,
			reader: reader,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("unwrap %s protocol: %w", protocol.GetName(), err)
		}
	}

	return nil, nil, fmt.Errorf("max unwrap depth reached")
}

func (a *AgentMux) determineProtocol(conn net.Conn) (mux.Protocol, io.Reader, error) {
	lg := a.lg
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	defer conn.SetReadDeadline(time.Time{})

	reader := bufio.NewReader(conn)
	header, err := reader.Peek(16)
	if err != nil {
		if err == io.EOF {
			return nil, nil, fmt.Errorf("connection closed")
		}
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, nil, fmt.Errorf("connection timed out")
		}
		return nil, nil, fmt.Errorf("unable to peek at connection: %w", err)
	}

	// protocol := protocols.GetProtocol(header)
	// if protocol == nil {
	// 	return nil, nil, fmt.Errorf("unknown protocol: %v", header)
	// }

	protocol := a.mux.GetProtocol(header)
	if protocol == nil {
		return nil, nil, fmt.Errorf("unknown protocol: %v", header)
	}

	lg.Debugf("Header: %v", header)
	lg.Debugf("Protocol: %s (unwrapped: %t)", protocol.GetName(), protocol.IsUnwrapped())

	return protocol, reader, nil
}
