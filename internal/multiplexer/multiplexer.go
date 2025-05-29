package multiplexer

import (
	"context"
	"fmt"
	"net"
	"rscc/internal/common/constants"
	"rscc/internal/common/logger"
	"rscc/internal/multiplexer/connection"
	"rscc/internal/multiplexer/listener"
	"rscc/internal/multiplexer/protocols"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/go-faster/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// MultiplexerConfig holds configuration to setup multiplexer
type MultiplexerConfig struct {
	// is HTTP download protocol can be used
	IsHttpDownload bool
	// is TCP download protocol can be used
	IsTcpDownload bool
	// is TLS transport can be used over HTTP
	IsTls bool
	// base directory path
	BasePath string
	// host to listen on
	Host string
	// Port to listen on
	Port int
	// Timeout for marking client as dead
	Timeout time.Duration
}

// Multiplexer holds related to multiplexer info
type Multiplexer struct {
	// root multiplexer logger
	lg     *zap.SugaredLogger
	config *MultiplexerConfig
	// single listener processing all multiplexer's requests
	listener net.Listener
	// mapper holds protocol and related listener
	mapper map[protocols.Proto]*listener.MultiplexerListener
	// queue holds clients'es connections
	queue chan net.Conn
}

// NewServer prepares multiplexer server
func NewServer(ctx context.Context, config *MultiplexerConfig) (*Multiplexer, error) {
	var m Multiplexer
	var err error

	m.config = config
	m.listener = nil
	m.mapper = make(map[protocols.Proto]*listener.MultiplexerListener, 0)
	m.queue = make(chan net.Conn)
	m.lg = logger.FromContext(ctx).Named("multiplexer")

	listenerConfig := net.ListenConfig{
		KeepAlive: m.config.Timeout,
	}
	if m.listener, err = listenerConfig.Listen(ctx, "tcp", net.JoinHostPort(m.config.Host, strconv.Itoa(m.config.Port))); err != nil {
		return nil, errors.Wrapf(err, "unable start new listener on %s", net.JoinHostPort(m.config.Host, strconv.Itoa(m.config.Port)))
	}
	m.lg.Infof("Listener started at %s", net.JoinHostPort(m.config.Host, strconv.Itoa(m.config.Port)))

	// Attach protocols on listener
	m.mapper[protocols.NewSshProto()] = listener.NewMultiplexerListener(m.listener.Addr(), protocols.Ssh)

	return &m, nil
}

// Start starts serving of multiplexer server
func (m *Multiplexer) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	// start accept loop for network connections on main listener
	g.Go(m.AcceptLoop)
	// start unwrapping loop on network connections from main listener
	g.Go(m.UnwrapLoop)
	g.Go(func() error {
		// wait for context cancelling or finishing
		<-ctx.Done()
		if err := m.Close(); err != nil {
			return err
		}
		m.lg.Info("Stop listener")
		return nil
	})

	return g.Wait()
}

// AcceptLoop accepts connections on main listener and send them to processor's queue
func (m *Multiplexer) AcceptLoop() error {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				return errors.Wrap(err, "accept connection on main listener")
			}
			return nil
		}
		// send connection to queue
		go func() {
			select {
			case m.queue <- conn:
			case <-time.After(2 * time.Second):
				m.lg.Warnf("Accept of new connection from %s timed out on main listener", conn.RemoteAddr().String())
				conn.Close()
			}
		}()
	}
}

// UnwrapLoop iterates over connections and unwrap them
func (m *Multiplexer) UnwrapLoop() error {
	var awaitingConnections int32

	for conn := range m.queue {
		// check excess of awaiting numbers
		if atomic.LoadInt32(&awaitingConnections) > constants.MaxNetworkClients {
			conn.Close()
			m.lg.Warnf("more then %d waiting connections opened. Dropping this one", awaitingConnections)
			continue
		}

		// increment awaiting number
		atomic.AddInt32(&awaitingConnections, 1)

		// unwrap protocol
		go func(conn net.Conn) {
			defer atomic.AddInt32(&awaitingConnections, -1)

			// unwrap data in connection
			newConn, proto, err := m.unwrap(conn)
			if err != nil {
				if conn != nil {
					conn.Close()
				}
				m.lg.Debugf("multiplex unwrap failed from %s: %v", conn.RemoteAddr().String(), err)
				return
			}

			// get multiplexer listener for protocol
			l, ok := m.mapper[proto]
			if !ok || l == nil {
				if newConn != nil {
					newConn.Close()
				}
				m.lg.Debugf("unknown multiplexer protocol from %s: %s", newConn.RemoteAddr().String(), proto.Type())
				return
			}

			// write connection to choosed listener
			select {
			case l.Queue() <- newConn:
			case <-time.After(2 * time.Second):
				if newConn != nil {
					newConn.Close()
				}
				m.lg.Warnf("Accept of new connection from %s timed out on protocol listener (%s)", conn.RemoteAddr().String(), proto.Type())
			}
		}(conn)
	}
	return nil
}

// unwrap unwraps data from connection to underlay protocol
func (m *Multiplexer) unwrap(conn net.Conn) (net.Conn, protocols.Proto, error) {
	// set deadline for waiting of first N bytes
	conn.SetDeadline(time.Now().Add(2 * time.Second))

	// initiali protocol determination
	conn, proto, err := m.determine(conn)
	if err != nil {
		return nil, protocols.NewUnknownProto(), errors.Wrap(err, "initial determination of protocol")
	}

	// reset deadline
	conn.SetDeadline(time.Time{})

	// if protocol already unwrapped
	if proto.IsUnwrapped() {
		return conn, proto, nil
	} else {
		// process next stage of procotol unwrapping
		// TODO
	}

	return nil, protocols.NewUnknownProto(), fmt.Errorf("nothing usable protocol was found for %s", proto.Type())
}

// determine determines protocol used for connection
func (m *Multiplexer) determine(conn net.Conn) (net.Conn, protocols.Proto, error) {
	header := make([]byte, constants.ConnHeaderLength)
	n, err := conn.Read(header)
	if err != nil {
		return nil, protocols.NewUnknownProto(), errors.Wrap(err, "read header bytes from connection")
	}

	// create buffered connection
	bufConn := connection.NewBufferedConn(header[:n], conn)

	// search for proto
	for k := range m.mapper {
		if k.IsProto(header) {
			return bufConn, k, nil
		}
	}

	return nil, protocols.NewUnknownProto(), fmt.Errorf("unknown protocol bytes: %v", header)
}

// Close closes multiplexer and related connections
func (m *Multiplexer) Close() error {
	// close queue with connections
	close(m.queue)
	// close multiplexer listeners
	for k, v := range m.mapper {
		if v != nil {
			if err := v.Close(); err != nil {
				m.lg.Warnf("close multiplexer listener for %s: %w", k, err)
			}
		}
	}
	// close main listener
	if m.listener != nil {
		return m.listener.Close()
	}
	return nil
}

// GetSshListener returns listener for SSH server
func (m *Multiplexer) GetSshListener() net.Listener {
	for k, v := range m.mapper {
		if k.Type() == protocols.Ssh {
			return v
		}
	}
	return nil
}
