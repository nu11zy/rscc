package http

import (
	"context"
	realhttp "net/http"
	"rscc/internal/common/network"
	"rscc/internal/database"
	"time"

	"go.uber.org/zap"
)

// TODO: Improve logging
type Protocol struct {
	queue        chan *network.BufferedConn
	listener     network.QueueListener
	server       *realhttp.Server
	fileServer   realhttp.Handler
	htmlPagePath string
	db           *database.Database
	lg           *zap.SugaredLogger
}

type ProtocolConfig struct {
	Db           *database.Database
	HtmlPagePath string
}

func NewProtocol(lg *zap.SugaredLogger, config *ProtocolConfig) (*Protocol, error) {
	lg = lg.Named("http")

	queue := make(chan *network.BufferedConn)
	listener := network.NewQueueListener(queue)

	protocol := &Protocol{
		queue:        queue,
		listener:     listener,
		htmlPagePath: config.HtmlPagePath,
		db:           config.Db,
		lg:           lg,
	}

	httpmux := realhttp.NewServeMux()
	httpmux.HandleFunc("/", protocol.RequestHandler)

	if protocol.htmlPagePath != "" {
		protocol.fileServer = realhttp.FileServer(realhttp.Dir(protocol.htmlPagePath))
	}

	protocol.server = &realhttp.Server{
		Handler: httpmux,
	}

	return protocol, nil
}

func (p *Protocol) GetName() string {
	return "http"
}

func (p *Protocol) GetHeader() [][]byte {
	return [][]byte{
		[]byte(realhttp.MethodConnect),
		[]byte(realhttp.MethodDelete),
		[]byte(realhttp.MethodGet),
		[]byte(realhttp.MethodHead),
		[]byte(realhttp.MethodOptions),
		[]byte(realhttp.MethodPatch),
		[]byte(realhttp.MethodPost),
		[]byte(realhttp.MethodPut),
		[]byte(realhttp.MethodTrace),
	}
}

func (p *Protocol) IsUnwrapped() bool {
	return true
}

func (p *Protocol) Unwrap(bufferedConn *network.BufferedConn) (*network.BufferedConn, error) {
	p.lg.Warn("HTTP protocol does not implement unwrap. Returning original connection")
	return bufferedConn, nil
}

func (p *Protocol) Handle(bufferedConn *network.BufferedConn) error {
	p.lg.Debugf("New HTTP connection from %s", bufferedConn.RemoteAddr())
	p.queue <- bufferedConn
	return nil
}

func (p *Protocol) StartListener(ctx context.Context) error {
	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := p.server.Shutdown(shutdownCtx); err != nil {
			p.lg.Errorf("Failed to shutdown HTTP listener: %v", err)
			return
		}

		p.lg.Warn("HTTP listener closed")
	}()

	p.lg.Info("HTTP listener started")
	err := p.server.Serve(p.listener)
	if err != nil && err != realhttp.ErrServerClosed {
		return err
	}

	return nil
}
