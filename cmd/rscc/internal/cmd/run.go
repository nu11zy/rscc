package cmd

import (
	"path/filepath"
	"rscc/internal/agentmux"
	"rscc/internal/common/logger"
	"rscc/internal/database"
	"rscc/internal/opsrv"
	"rscc/internal/session"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func (c *Cmd) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	lg := logger.FromContext(ctx)

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
		Db:       db,
		Sm:       sm,
		Host:     c.OperatorHost,
		Port:     c.OperatorPort,
		DataPath: c.DataPath,
	}
	opsrv, err := opsrv.NewServer(ctx, opsrvParams)
	if err != nil {
		lg.Errorf("Failed to initialize operator server: %v", err)
		return err
	}

	// Create agent mux
	agentMuxParams := &agentmux.AgentMuxParams{
		Host:     c.AgentHost,
		Port:     c.AgentPort,
		DataPath: c.DataPath,
	}
	agentMux, err := agentmux.NewAgentMux(ctx, agentMuxParams)
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
