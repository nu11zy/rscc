package network

import (
	"bufio"
	"net"
	"time"
)

type TimeoutConn struct {
	net.Conn
	Timeout time.Duration
}

// NewTimeoutConn creates new TimeoutConn
func NewTimeoutConn(conn net.Conn, timeout time.Duration) *TimeoutConn {
	return &TimeoutConn{
		Conn:    conn,
		Timeout: timeout,
	}
}

// SetTimeout sets timeout for connection
func (t *TimeoutConn) SetTimeout(timeout time.Duration) {
	t.Timeout = timeout
}

// Read reads data from connection with deadline
func (t *TimeoutConn) Read(b []byte) (int, error) {
	if t.Timeout != 0 {
		t.Conn.SetDeadline(time.Now().Add(t.Timeout))
	}
	return t.Conn.Read(b)
}

// Write writes data to connection with deadline
func (t *TimeoutConn) Write(b []byte) (int, error) {
	if t.Timeout != 0 {
		t.Conn.SetDeadline(time.Now().Add(t.Timeout))
	}
	return t.Conn.Write(b)
}

type BufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func NewBufferedConn(conn net.Conn) *BufferedConn {
	return &BufferedConn{
		Conn:   conn,
		reader: bufio.NewReader(conn),
	}
}

func (c *BufferedConn) Read(b []byte) (int, error) {
	return c.reader.Read(b)
}

func (c *BufferedConn) Peek(n int) ([]byte, error) {
	return c.reader.Peek(n)
}
