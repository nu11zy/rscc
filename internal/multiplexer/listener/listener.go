package listener

import (
	"net"
	"rscc/internal/multiplexer/protocols"
)

// MultiplexerListener implements interface for net.Listener
type MultiplexerListener struct {
	addr   net.Addr
	queue  chan net.Conn
	closed bool
	proto  protocols.Type
}

// NetMultiplexerListeners returns object of multiplexer's listener
func NewMultiplexerListener(addr net.Addr, proto protocols.Type) *MultiplexerListener {
	return &MultiplexerListener{
		addr:   addr,
		queue:  make(chan net.Conn),
		closed: false,
		proto:  proto,
	}
}

// Queue returns chan with connection
func (m *MultiplexerListener) Queue() chan net.Conn {
	return m.queue
}

// Accept will wait for new object of net.Conn in chan queue
func (m *MultiplexerListener) Accept() (net.Conn, error) {
	if m.closed {
		return nil, net.ErrClosed
	}
	// this check must be for handling close(m.queue) in Close() function
	// otherwise reciever get (nil, nil) and will be panic (e.g. on SSH/HTTP servers)
	conn := <-m.queue
	if conn == nil {
		return nil, net.ErrClosed
	}
	return conn, nil
}

// Close closes multiplexer listener
func (m *MultiplexerListener) Close() error {
	if !m.closed {
		m.closed = true
		close(m.queue)
	}
	return nil
}

// Addr returns listener's network address
func (m *MultiplexerListener) Addr() net.Addr {
	if m.closed {
		return nil
	}
	return m.addr
}
