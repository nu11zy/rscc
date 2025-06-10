package agentsrv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"rscc/internal/agentsrv/mux"
	"rscc/internal/agentsrv/mux/http"
	"rscc/internal/agentsrv/mux/ssh"
	"rscc/internal/agentsrv/mux/tls"
	"rscc/internal/common/constants"
	"rscc/internal/common/logger"
	"rscc/internal/common/network"
	"rscc/internal/database"
	"rscc/internal/session"
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
	Host         string
	Port         int
	DataPath     string
	TlsCertPath  string
	TlsKeyPath   string
	HtmlPagePath string
	Db           *database.Database
	Sm           *session.SessionManager
}

func NewAgentMux(ctx context.Context, params *AgentMuxParams) (*AgentMux, error) {
	lg := logger.FromContext(ctx).Named("agent")

	address := net.JoinHostPort(params.Host, strconv.Itoa(params.Port))

	muxConfig := &mux.MuxConfig{
		TlsConfig: &tls.ProtocolConfig{
			TlsCertPath: params.TlsCertPath,
			TlsKeyPath:  params.TlsKeyPath,
		},
		HttpConfig: &http.ProtocolConfig{
			Db:           params.Db,
			HtmlPagePath: params.HtmlPagePath,
		},
		SshConfig: &ssh.ProtocolConfig{
			Db: params.Db,
			Sm: params.Sm,
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
		lg.Warn("Agent listener closed")
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

				bufferedConn := network.NewBufferedConn(conn)
				protocol, unwrappedConn, err := a.unwrapConnection(bufferedConn)
				if err != nil {
					lg.Errorf("Failed to unwrap connection: %v", err)
					bufferedConn.Close()
					return
				}

				if err := protocol.Handle(unwrappedConn); err != nil {
					lg.Errorf("Failed to handle connection: %v", err)
					unwrappedConn.Close()
					bufferedConn.Close()
					return
				}
			}(conn)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (a *AgentMux) unwrapConnection(bufferedConn *network.BufferedConn) (mux.Protocol, *network.BufferedConn, error) {
	lg := a.lg

	for i := 0; i < constants.MaxUnwrapDepth; i++ {
		protocol, err := a.determineProtocol(bufferedConn)
		if err != nil {
			return nil, nil, fmt.Errorf("determine protocol: %w", err)
		}

		if protocol.IsUnwrapped() {
			lg.Debugf("Successfully unwrapped %s protocol from %s", protocol.GetName(), bufferedConn.RemoteAddr())
			return protocol, bufferedConn, nil
		}

		bufferedConn, err = protocol.Unwrap(bufferedConn)
		if err != nil {
			return nil, nil, fmt.Errorf("unwrap %s protocol: %w", protocol.GetName(), err)
		}
	}

	return nil, nil, fmt.Errorf("max unwrap depth reached")
}

func (a *AgentMux) determineProtocol(bufferedConn *network.BufferedConn) (mux.Protocol, error) {
	lg := a.lg

	bufferedConn.SetReadDeadline(time.Now().Add(time.Second * 5))
	defer bufferedConn.SetReadDeadline(time.Time{})

	header, err := bufferedConn.Peek(16)
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("connection closed")
		}
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, fmt.Errorf("connection timed out")
		}
		return nil, fmt.Errorf("unable to peek at connection: %w", err)
	}

	protocol := a.mux.GetProtocol(header)
	if protocol == nil {
		return nil, fmt.Errorf("unknown protocol: %v", header)
	}

	lg.Debugf("Header: %v", header)
	lg.Debugf("Protocol: %s (unwrapped: %t)", protocol.GetName(), protocol.IsUnwrapped())

	return protocol, nil
}
