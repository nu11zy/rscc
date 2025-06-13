package cmd

import (
	"net"
	"path/filepath"
	"rscc/internal/agentsrv"
	"rscc/internal/common/logger"
	"rscc/internal/database"
	"rscc/internal/opsrv"
	"rscc/internal/session"
	"strconv"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func (c *Cmd) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	lg := logger.FromContext(ctx)

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
		Address:      agentAddr,
		DataPath:     c.DataPath,
		HtmlPagePath: c.HtmlPagePath,
		Db:           db,
		Sm:           sm,
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
