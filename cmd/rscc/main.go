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

	appRoot := &cobra.Command{
		Use:               "rscc [command]",
		Short:             "Reverse SSH command & control",
		PersistentPreRunE: preRun,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		SilenceUsage: true,
	}
	appRoot.PersistentFlags().StringVar(&dbPath, "db", "rscc.db", "database path")
	appRoot.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug mode")

	appAdmin := &cobra.Command{
		Use:   "admin [flags]",
		Short: "Create new admin",
		RunE:  adminCmd,
	}
	appAdmin.Flags().StringVarP(&operatorName, "name", "n", "", "operator name")
	appAdmin.Flags().StringVarP(&publicKey, "key", "k", "", "operator public key")
	appAdmin.MarkFlagRequired("name")
	appAdmin.MarkFlagRequired("key")
	appRoot.AddCommand(appAdmin)

	appStart := &cobra.Command{
		Use:   "start [flags]",
		Short: "Start rscc",
		RunE:  startCmd,
	}
	appStart.Flags().IntVar(&operatorPort, "op", 55022, "operator listener port")
	appStart.Flags().StringVar(&operatorHost, "oh", "0.0.0.0", "operator listener host")
	appStart.Flags().IntVar(&agentPort, "ap", 5522, "agent listener port")
	appStart.Flags().StringVar(&agentHost, "ah", "0.0.0.0", "agent listener host")
	appRoot.AddCommand(appStart)

	if err := appRoot.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func preRun(cmd *cobra.Command, args []string) error {
	if debug {
		logger.SetDebug()
	}
	return nil
}

func adminCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	lg := logger.FromContext(ctx)

	db, err := database.NewDatabase(ctx, dbPath)
	if err != nil {
		lg.Errorf("Failed to initialize database: %v", err)
		return err
	}

	user, err := db.CreateOperator(ctx, operatorName, publicKey, true)
	if err != nil {
		lg.Errorf("Failed to add operator: %v", err)
		return err
	}

	lg.Infof("New admin operator `%s` with id `%s` created", user.Name, user.ID)
	return nil
}

func startCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	lg := logger.FromContext(ctx)

	// Initialize database
	db, err := database.NewDatabase(ctx, dbPath)
	if err != nil {
		lg.Errorf("Failed to initialize database: %v", err)
		return err
	}

	// Check if at least one operator exists
	operators, err := db.GetAllOperators(ctx)
	if err != nil {
		lg.Errorf("Failed to get all operators: %v", err)
		return err
	}
	if len(operators) == 0 {
		lg.Errorf("Admin operator not found. Use `rscc admin` to create new admin operator.")
		return fmt.Errorf("admin operator not found")
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
