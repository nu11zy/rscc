package mux

import (
	"bytes"
	"context"
	"fmt"
	"rscc/internal/agentsrv/mux/http"
	"rscc/internal/agentsrv/mux/ssh"
	"rscc/internal/agentsrv/mux/tcp"
	"rscc/internal/agentsrv/mux/tls"
	"rscc/internal/common/network"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Protocol interface {
	GetName() string
	GetHeader() [][]byte
	IsUnwrapped() bool
	Unwrap(conn *network.BufferedConn) (*network.BufferedConn, error)
	Handle(conn *network.BufferedConn) error
	StartListener(ctx context.Context) error
}

type Mux struct {
	lg        *zap.SugaredLogger
	protocols []Protocol
}

type MuxConfig struct {
	TlsConfig  *tls.ProtocolConfig
	HttpConfig *http.ProtocolConfig
}

func NewMux(lg *zap.SugaredLogger, config *MuxConfig) (*Mux, error) {
	lg = lg.Named("mux")

	tcpProtocol, err := tcp.NewProtocol(lg)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP protocol: %w", err)
	}

	sshProtocol, err := ssh.NewProtocol(lg)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH protocol: %w", err)
	}

	tlsProtocol, err := tls.NewProtocol(lg, config.TlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS protocol: %w", err)
	}

	httpProtocol, err := http.NewProtocol(lg, config.HttpConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP protocol: %w", err)
	}

	return &Mux{
		lg:        lg,
		protocols: []Protocol{tcpProtocol, sshProtocol, tlsProtocol, httpProtocol},
	}, nil
}

func (m *Mux) GetProtocol(data []byte) Protocol {
	for _, protocol := range m.protocols {
		for _, header := range protocol.GetHeader() {
			if bytes.HasPrefix(data, header) {
				return protocol
			}
		}
	}
	return nil
}

func (m *Mux) Start(ctx context.Context) error {
	m.lg.Info("Starting mux")
	g, ctx := errgroup.WithContext(ctx)

	for _, protocol := range m.protocols {
		if protocol.IsUnwrapped() {
			g.Go(func() error {
				return protocol.StartListener(ctx)
			})
		}
	}

	return g.Wait()
}
