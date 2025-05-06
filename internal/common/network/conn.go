package network

import (
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
