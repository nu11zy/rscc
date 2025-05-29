package cmd

import (
	"fmt"
	"path/filepath"
	"rscc/internal/common/logger"
	"rscc/internal/database"
	"rscc/internal/multiplexer"
	"rscc/internal/multiplexer/http"
	"rscc/internal/multiplexer/ssh"
	"rscc/internal/opsrv"
	"rscc/internal/session"
	"time"

	"github.com/go-faster/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func (c *Cmd) Run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	lg := logger.FromContext(ctx)

	// initialize database
	db, err := database.NewDatabase(ctx, filepath.Join(c.DataDirectory, "rscc.db"))
	if err != nil {
		lg.Errorf("Failed to initialize database: %v", err)
		return err
	}

	// create session manager
	sm := session.NewSessionManager(ctx, db)

	// operator listener
	operatorData := &opsrv.OperatorServerConfig{
		Host:     c.OperatorHost,
		Port:     int(c.OperatorPort),
		BasePath: c.DataDirectory,
	}
	opSrv, err := opsrv.NewServer(ctx, db, sm, operatorData)
	if err != nil {
		return errors.Wrap(err, "failed to initialize operator's server")
	}

	// multiplexer listener
	multiplexerData := &multiplexer.MultiplexerConfig{
		Host:                c.MultiplexerHost,
		Port:                int(c.MultiplexerPort),
		BasePath:            c.DataDirectory,
		HttpDownloadEnabled: c.HttpDownload,
		TcpDownloadEnabled:  c.TcpDownload,
		TlsEnabled:          c.TlsEnabled,
		TlsCertificate:      c.TlsCertificate,
		Timeout:             time.Duration(time.Duration(c.Timeout) * time.Second),
	}
	muxSrv, err := multiplexer.NewServer(ctx, multiplexerData)
	if err != nil {
		return errors.Wrap(err, "failed to initialize multiplexer's server")
	}

	// ssh agent server
	if muxSrv.GetSshListener() == nil {
		return fmt.Errorf("no suitable listener found for agent SSH server")
	}
	sshConfig := &ssh.SshConfig{
		Listener: muxSrv.GetSshListener(),
		Timeout:  time.Duration(time.Duration(c.Timeout) * time.Second),
	}
	sshSrv, err := ssh.NewServer(ctx, db, sm, sshConfig)
	if err != nil {
		return errors.Wrap(err, "failed to initialize agent's SSH server")
	}

	// http agent server
	var httpSrv *http.Http
	if muxSrv.GetHttpListener() == nil {
		lg.Warn("No HTTP server will be served")
	} else {
		httpConfig := &http.HttpConfig{
			Listener:       muxSrv.GetHttpListener(),
			Timeout:        time.Duration(time.Duration(c.Timeout) * time.Second),
			IsHttpDownload: c.HttpDownload,
			PlugPageBytes:  c.HttpPlugPageBytes,
			PlugPageCode:   c.HttpPlugPageCode,
		}
		httpSrv, err = http.NewServer(ctx, db, httpConfig)
		if err != nil {
			return errors.Wrap(err, "failed to initialize agent's HTTP server")
		}
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return opSrv.Start(ctx) })
	g.Go(func() error { return muxSrv.Start(ctx) })
	g.Go(func() error { return sshSrv.Start(ctx) })
	if httpSrv != nil {
		g.Go(func() error { return httpSrv.Start(ctx) })
	}
	return g.Wait()
}
