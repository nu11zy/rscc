package protocols

import (
	"crypto/tls"
	"fmt"
	"net"
)

var tlsProtocol = Protocol{
	Name:        "tls",
	Header:      [][]byte{{0x16, 0x03, 0x01}},
	IsUnwrapped: false,
	Unwrap:      Unwrap,
}

func init() {
	protocols = append(protocols, tlsProtocol)
}

func Unwrap(conn net.Conn) (net.Conn, error) {
	tlsConn := tls.Server(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return nil, fmt.Errorf("tls handshake failed: %w", err)
	}
	return tlsConn, nil
}
