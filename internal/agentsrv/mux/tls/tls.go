package tls

import (
	"context"
	"crypto/tls"
	"fmt"
	"rscc/internal/common/network"
	"rscc/internal/common/utils"

	"go.uber.org/zap"
)

type Protocol struct {
	lg        *zap.SugaredLogger
	tlsConfig *tls.Config
}

type ProtocolConfig struct {
	TlsCertPath string
	TlsKeyPath  string
}

func NewProtocol(lg *zap.SugaredLogger, config *ProtocolConfig) (*Protocol, error) {
	lg = lg.Named("tls")

	var err error
	var cert tls.Certificate
	if config.TlsCertPath != "" && config.TlsKeyPath != "" {
		cert, err = tls.LoadX509KeyPair(config.TlsCertPath, config.TlsKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
		}
		lg.Infof("Using TLS certificate from %s", config.TlsCertPath)
	} else { // generate self-signed certificate
		cert, err = utils.GenTlsCertificate("127.0.0.1")
		if err != nil {
			return nil, fmt.Errorf("failed to generate self-signed certificate: %w", err)
		}
		lg.Warnf("No TLS certificate provided, using self-signed certificate")
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS10,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}
	tlsConfig.Certificates = append(tlsConfig.Certificates, cert)

	return &Protocol{
		lg:        lg,
		tlsConfig: tlsConfig,
	}, nil
}

func (p *Protocol) GetName() string {
	return "tls"
}

func (p *Protocol) GetHeader() [][]byte {
	return [][]byte{{0x16, 0x03, 0x01}}
}

func (p *Protocol) IsUnwrapped() bool {
	return false
}

func (p *Protocol) Unwrap(bufferedConn *network.BufferedConn) (*network.BufferedConn, error) {
	p.lg.Debugf("Unwrapping TLS connection from %s", bufferedConn.RemoteAddr())
	tlsConn := tls.Server(bufferedConn, p.tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return nil, fmt.Errorf("tls handshake failed: %w", err)
	}
	p.lg.Debugf("TLS handshake successful")
	return network.NewBufferedConn(tlsConn), nil
}

func (p *Protocol) Handle(conn *network.BufferedConn) error {
	return fmt.Errorf("tls protocol does not implement handling")
}

func (p *Protocol) StartListener(ctx context.Context) error {
	return nil
}
