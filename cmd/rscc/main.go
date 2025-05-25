package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"rscc/internal/common/logger"
	"rscc/internal/database"
	"rscc/internal/listener"
	"rscc/internal/opsrv"
	"rscc/internal/session"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	operatorPort int
	operatorHost string
	agentPort    int
	agentHost    string
	operatorName string
	publicKey    string
	dbPath       string
	debug        bool
)

type Cmd struct{}

func main() {
	// Initialize context and logger
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	lg, err := logger.NewLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer lg.Sync()
	ctx = logger.WithLogger(ctx, lg)

	// Initialize root command
	root := &cobra.Command{
		Use:     "rscc",
		Short:   "Reverse SSH command & control",
		PreRunE: preRun,
		RunE:    run,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		SilenceUsage: true,
	}
	root.Flags().IntVar(&operatorPort, "op", 55022, "operator listener port")
	root.Flags().StringVar(&operatorHost, "oh", "0.0.0.0", "operator listener host")
	root.Flags().IntVar(&agentPort, "ap", 8080, "agent listener port")
	root.Flags().StringVar(&agentHost, "ah", "0.0.0.0", "agent listener host")
	root.Flags().StringVar(&dbPath, "db", "rscc.db", "database path")
	root.Flags().BoolVar(&debug, "debug", false, "enable debug mode")

	if err := root.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func preRun(cmd *cobra.Command, args []string) error {
	if debug {
		logger.SetDebug()
	}
	return nil
}

func run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	lg := logger.FromContext(ctx)

	// Initialize database
	db, err := database.NewDatabase(ctx, dbPath)
	if err != nil {
		lg.Errorf("Failed to initialize database: %v", err)
		return err
	}

	// Start session manager
	sm := session.NewSessionManager(ctx, db)

	// Start operator listener
	opsrv, err := opsrv.NewOperatorServer(ctx, db, sm, operatorHost, operatorPort)
	if err != nil {
		lg.Errorf("Failed to initialize operator server: %v", err)
		return err
	}

	// Start agent listener
	agentListener, err := listener.NewAgentListener(ctx, db, sm, agentHost, agentPort)
	if err != nil {
		lg.Errorf("Failed to initialize agent listener: %v", err)
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return opsrv.Start(ctx) })
	g.Go(func() error { return agentListener.Start(ctx) })
	return g.Wait()
}
