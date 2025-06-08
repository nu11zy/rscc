package http

import (
	"context"
	"net"
	"net/http"

	"go.uber.org/zap"
)

type Protocol struct {
	queue chan net.Conn
	lg    *zap.SugaredLogger
}

func NewProtocol(lg *zap.SugaredLogger) (*Protocol, error) {
	lg = lg.Named("http")

	return &Protocol{
		queue: make(chan net.Conn, 128),
		lg:    lg,
	}, nil
}

func (p *Protocol) GetName() string {
	return "http"
}

func (p *Protocol) GetHeader() [][]byte {
	return [][]byte{
		[]byte(http.MethodConnect),
		[]byte(http.MethodDelete),
		[]byte(http.MethodGet),
		[]byte(http.MethodHead),
		[]byte(http.MethodOptions),
		[]byte(http.MethodPatch),
		[]byte(http.MethodPost),
		[]byte(http.MethodPut),
		[]byte(http.MethodTrace),
	}
}

func (p *Protocol) IsUnwrapped() bool {
	return true
}

func (p *Protocol) Unwrap(conn net.Conn) (net.Conn, error) {
	p.lg.Warn("HTTP protocol does not implement unwrap. Returning original connection")
	return conn, nil
}

func (p *Protocol) Handle(conn net.Conn) error {
	p.lg.Debugf("New HTTP connection from %s", conn.RemoteAddr())
	p.queue <- conn
	return nil
}

func (p *Protocol) HandleLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case conn := <-p.queue:
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

			conn.Close()
			p.lg.Debugf("Closed HTTP connection from %s", conn.RemoteAddr())
		}
	}
}
