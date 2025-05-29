package connection

import (
	"net"
	"time"
)

type BufferedConn struct {
	// prefix is already read data from begin of stream
	prefix []byte
	conn   net.Conn
}

// NewBufferedConn creates new connection with already read prefix
func NewBufferedConn(prefix []byte, conn net.Conn) *BufferedConn {
	return &BufferedConn{
		prefix: prefix,
		conn:   conn,
	}
}

func (b *BufferedConn) Read(data []byte) (int, error) {
	if len(b.prefix) > 0 {
		n := copy(data, b.prefix)
		b.prefix = b.prefix[n:]
		return n, nil
	}
	return b.conn.Read(data)
}

func (b *BufferedConn) Write(data []byte) (int, error) {
	return b.conn.Write(data)
}

func (b *BufferedConn) Close() error {
	return b.conn.Close()
}

func (b *BufferedConn) LocalAddr() net.Addr {
	return b.conn.LocalAddr()
}

func (b *BufferedConn) RemoteAddr() net.Addr {
	return b.conn.RemoteAddr()
}

func (b *BufferedConn) SetDeadline(t time.Time) error {
	return b.conn.SetDeadline(t)
}

func (b *BufferedConn) SetReadDeadline(t time.Time) error {
	return b.conn.SetReadDeadline(t)
}

func (b *BufferedConn) SetWriteDeadline(t time.Time) error {
	return b.conn.SetWriteDeadline(t)
}
