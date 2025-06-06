package protocols

import (
	"bytes"
	"crypto/tls"
	"net"
)

var tlsConfig *tls.Config
var protocols = []Protocol{}

type Protocol struct {
	Name        string
	Header      [][]byte
	IsUnwrapped bool
	Unwrap      func(conn net.Conn) (net.Conn, error)
}

func GetProtocol(data []byte) *Protocol {
	for _, protocol := range protocols {
		for _, header := range protocol.Header {
			if bytes.HasPrefix(data, header) {
				return &protocol
			}
		}
	}
	return nil
}

func SetTlsCertificate(cert tls.Certificate) {
	tlsConfig = &tls.Config{}
	tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
}
