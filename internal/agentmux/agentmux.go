package agentmux

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"rscc/internal/agentmux/protocols"
	"rscc/internal/common/constants"
	"rscc/internal/common/logger"
	"rscc/internal/common/utils"
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
	lg := logger.FromContext(ctx).Named("mux")

	address := net.JoinHostPort(params.Host, strconv.Itoa(params.Port))

	var err error
	var cert tls.Certificate
	if params.TlsCertPath != "" && params.TlsKeyPath != "" {
		cert, err = tls.LoadX509KeyPair(params.TlsCertPath, params.TlsKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
		}
	} else { // generate self-signed certificate
		lg.Warnf("No TLS certificate provided, generating self-signed certificate")
		cert, err = utils.GenTlsCertificate("127.0.0.1")
		if err != nil {
			return nil, fmt.Errorf("failed to generate self-signed certificate: %w", err)
		}
	}
	protocols.SetTlsCertificate(cert)

	return &AgentMux{
		connQueue: make(chan net.Conn),
		connSem:   semaphore.NewWeighted(constants.MaxUnwrapConnections),
		address:   address,
		dataPath:  params.DataPath,
		lg:        lg,
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
				// Simulate a long-running connection
				// TODO: Pass unwrapped connection to the handler for the protocol
				_, _, err := a.unwrapConnection(conn)
				if err != nil {
					lg.Errorf("Failed to unwrap connection: %v", err)
					conn.Close()
					a.connSem.Release(1)
					return
				}
				time.Sleep(time.Second * 10)
				lg.Debugf("Closing connection from %s", conn.RemoteAddr())
				conn.Close()
				a.connSem.Release(1)
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

func (a *AgentMux) unwrapConnection(conn net.Conn) (*protocols.Protocol, net.Conn, error) {
	lg := a.lg

	for i := 0; i < constants.MaxUnwrapDepth; i++ {
		protocol, reader, err := a.determineProtocol(conn)
		if err != nil {
			return nil, nil, fmt.Errorf("determine protocol: %w", err)
		}

		if protocol.IsUnwrapped {
			lg.Debugf("Successfully unwrapped %s protocol from %s", protocol.Name, conn.RemoteAddr())
			return protocol, conn, nil
		}

		if protocol.Unwrap != nil {
			lg.Debugf("Unwrapping %s protocol from %s", protocol.Name, conn.RemoteAddr())
			conn, err = protocol.Unwrap(&peekableConn{
				Conn:   conn,
				reader: reader,
			})
			if err != nil {
				return nil, nil, fmt.Errorf("unwrap %s protocol: %w", protocol.Name, err)
			}
		} else {
			return nil, nil, fmt.Errorf("unwrapping %s protocol is not implemented", protocol.Name)
		}
	}

	return nil, nil, fmt.Errorf("max unwrap depth reached")
}

func (a *AgentMux) determineProtocol(conn net.Conn) (*protocols.Protocol, io.Reader, error) {
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

	protocol := protocols.GetProtocol(header)
	if protocol == nil {
		return nil, nil, fmt.Errorf("unknown protocol: %v", header)
	}

	lg.Debugf("Header: %v", header)
	lg.Debugf("Protocol: %s (unwrapped: %t)", protocol.Name, protocol.IsUnwrapped)

	return protocol, reader, nil
}
