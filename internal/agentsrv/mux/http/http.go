package http

import (
	"net"
	"net/http"

	"go.uber.org/zap"
)

type Protocol struct {
	lg *zap.SugaredLogger
}

func NewProtocol(lg *zap.SugaredLogger) (*Protocol, error) {
	lg = lg.Named("http")

	return &Protocol{
		lg: lg,
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
