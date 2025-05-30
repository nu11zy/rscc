package http

import (
	"context"
	"fmt"
	"io"
	"net"
	realhttp "net/http"
	"os"
	"path/filepath"
	"rscc/internal/common/logger"
	"rscc/internal/database"
	"rscc/internal/database/ent"
	"strings"
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
	mux.HandleFunc("/", http.DefaultMux)

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

// DefaultMux is default processor for HTTP server. Used for HTTP serving and plugging
func (h *Http) DefaultMux(w realhttp.ResponseWriter, r *realhttp.Request) {
	lg := h.lg.Named("default")
	lg.Infof("[%s] %s - %s", r.Host, r.RemoteAddr, r.URL.Path)

	// is HTTP download enabled -> process
	if h.config.IsHttpDownload {
		if err := h.httpDownloadRequest(w, r); err != nil {
			lg.Errorf("Process HTTP download request: %v", err)
			// write plug page in case of any errors
			if err := h.plugPageWriter(w); err != nil {
				lg.Error(err.Error())
			}
		}
		return
	}

	// return plug
	if err := h.plugPageWriter(w); err != nil {
		lg.Error(err.Error())
	}
}

// httpDownloadRequest processes delivery of agent's binaries via HTTP
func (h *Http) httpDownloadRequest(w realhttp.ResponseWriter, r *realhttp.Request) error {
	var agent *ent.Agent
	var err error
	filename := r.URL.Path
	extension := filepath.Ext(filename)
	filenameNoExt := strings.TrimSuffix(filename, extension)

	// get agent by url from DB
	if agent, err = h.db.GetAgentByUrl(r.Context(), filename); err != nil {
		if ent.IsNotFound(err) {
			if agent, err = h.db.GetAgentByUrl(r.Context(), filenameNoExt); err != nil {
				if !ent.IsNotFound(err) {
					return errors.Wrap(err, "get agent by url from DB")
				}
				return h.plugPageWriter(w)
			}
		}
	}

	// TODO: templates based on extensions

	// open FD to agent's file
	file, err := os.Open(agent.Path)
	if err != nil {
		return errors.Wrapf(err, "unable open agent's binary by path %s", agent.Path)
	}
	defer file.Close()

	// update hits number
	if err = h.db.UpdateAgentHits(r.Context(), agent.ID); err != nil {
		return errors.Wrapf(err, "unable update agent %s hits number for binary", agent.ID)
	}

	// write binary to client
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", agent.Name))
	w.Header().Set("Content-Type", "application/octet-stream")
	// ignore both result and errors
	io.Copy(w, file)
	return nil
}

// plugPageWriter writes predefined plug page with custom status code on response
func (h *Http) plugPageWriter(w realhttp.ResponseWriter) error {
	w.Header().Set("content-type", "text/html")
	w.Header().Set("server", "nginx")
	w.Header().Set("connection", "keep-alive")
	w.WriteHeader(h.config.PlugPageCode)
	_, err := w.Write(h.config.PlugPageBytes)
	return err
}
