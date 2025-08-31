package cmd

import (
	"net"
	"path/filepath"
	"strconv"

	"github.com/nu11zy/rscc/internal/agentsrv"
	"github.com/nu11zy/rscc/internal/common/logger"
	"github.com/nu11zy/rscc/internal/common/version"
	"github.com/nu11zy/rscc/internal/database"
	"github.com/nu11zy/rscc/internal/opsrv"
	"github.com/nu11zy/rscc/internal/session"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func (c *Cmd) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	lg := logger.FromContext(ctx)

	lg.Infof("Starting rscc %s", version.Full())

	operatorAddr := net.JoinHostPort(c.OperatorHost, strconv.Itoa(c.OperatorPort))
	agentAddr := net.JoinHostPort(c.AgentHost, strconv.Itoa(c.AgentPort))

	// Initialize database
	db, err := database.NewDatabase(ctx, filepath.Join(c.DataPath, "rscc.db"))
	if err != nil {
		lg.Errorf("Failed to initialize database: %v", err)
		return err
	}

	// Create session manager
	sm := session.NewSessionManager(ctx, db)

	// Create operator server
	opsrvParams := &opsrv.OperatorServerParams{
		Db:              db,
		Sm:              sm,
		OperatorAddress: operatorAddr,
		AgentAddress:    agentAddr,
		DataPath:        c.DataPath,
	}
	opsrv, err := opsrv.NewServer(ctx, opsrvParams)
	if err != nil {
		lg.Errorf("Failed to initialize operator server: %v", err)
		return err
	}

	// Create agent mux
	agentMuxParams := &agentsrv.AgentMuxParams{
		Address:        agentAddr,
		DataPath:       c.DataPath,
		HtmlPagePath:   c.HtmlPagePath,
		Db:             db,
		Sm:             sm,
		TrustAnySshKey: c.AgentTrust,
	}
	agentMux, err := agentsrv.NewAgentMux(ctx, agentMuxParams)
	if err != nil {
		lg.Errorf("Failed to initialize agent mux: %v", err)
		return err
	}

	// Start
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return opsrv.Start(ctx) })
	g.Go(func() error { return agentMux.Start(ctx) })
	return g.Wait()
}
