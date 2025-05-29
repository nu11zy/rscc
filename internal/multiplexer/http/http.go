package http

import (
	"context"
	"net"
	realhttp "net/http"
	"rscc/internal/common/logger"
	"rscc/internal/database"
	"time"

	"github.com/go-faster/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type HttpConfig struct {
	Listener       net.Listener
	Timeout        time.Duration
	PlugPageBytes  []byte
	PlugPageCode   int
	IsHttpDownload bool
}

type Http struct {
	lg       *zap.SugaredLogger
	db       *database.Database
	config   *HttpConfig
	srv      *realhttp.Server
	listener net.Listener
}

// NewServer prepares environment for new HTTP server
func NewServer(ctx context.Context, db *database.Database, config *HttpConfig) (*Http, error) {
	lg := logger.FromContext(ctx).Named("http")

	http := &Http{
		lg:       lg,
		db:       db,
		config:   config,
		listener: config.Listener,
		srv: &realhttp.Server{
			ReadTimeout:  config.Timeout,
			WriteTimeout: config.Timeout,
			Handler:      nil,
		},
	}

	mux := realhttp.NewServeMux()
	// default mux for processing
	mux.HandleFunc("/", http.Default)

	http.srv.Handler = mux
	return http, nil
}

// Start starts HTTP server
func (h *Http) Start(ctx context.Context) error {
	h.lg.Infof("Start HTTP server")

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if err := h.srv.Serve(h.listener); err != nil {
			if !errors.Is(err, net.ErrClosed) && !errors.Is(err, realhttp.ErrServerClosed) {
				return errors.Wrap(err, "error on HTTP server")
			}
		}
		return nil
	})

	g.Go(func() error {
		<-ctx.Done()
		if err := h.Stop(); err != nil {
			h.lg.Warn("Stop server: %v", err)
		}
		h.lg.Info("Stop server")
		return nil
	})
	return g.Wait()
}

// Stop closes listener if it's active
func (s *Http) Stop() error {
	if s.srv != nil {
		return s.srv.Shutdown(context.TODO())
	}
	return nil
}

// Default is default processor for HTTP server. Used for HTTP serving and plugging
func (h *Http) Default(w realhttp.ResponseWriter, r *realhttp.Request) {
	lg := h.lg.Named("default")
	if err := h.plugPageWriter(w); err != nil {
		lg.Error(err.Error())
	}
}

func (h *Http) plugPageWriter(w realhttp.ResponseWriter) error {
	w.Header().Set("content-type", "text/html")
	w.Header().Set("server", "nginx")
	w.Header().Set("connection", "keep-alive")
	w.WriteHeader(h.config.PlugPageCode)
	_, err := w.Write(h.config.PlugPageBytes)
	return err
}
