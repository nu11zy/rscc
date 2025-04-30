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

	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

var (
	operatorPort     int
	operatorHost     string
	agentPort        int
	agentHost        string
	operatorUsername string
	publicKey        string
	dbPath           string
	debug            bool
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

	// Initialize CLI app
	app := &cli.Command{
		Name:      "rscc",
		Usage:     "reverse SSH command & control",
		UsageText: "rscc [flags]",
		Commands: []*cli.Command{
			{
				Name:      "admin",
				Usage:     "Create new admin",
				UsageText: "rscc admin [flags]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "u",
						Usage:       "admin `USERNAME`",
						Destination: &operatorUsername,
						Required:    true,
					},
					&cli.StringFlag{
						Name:        "k",
						Usage:       "admin `PUBLIC_KEY`",
						Destination: &publicKey,
						Required:    true,
					},
				},
				Action: addAdmin,
			},
			{
				Name:      "start",
				Usage:     "Start rscc",
				UsageText: "rscc start [flags]",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:        "op",
						Usage:       "operator listener `PORT`",
						Destination: &operatorPort,
						Value:       55022,
					},
					&cli.StringFlag{
						Name:        "oh",
						Usage:       "operator listener `HOST`",
						Destination: &operatorHost,
						Value:       "0.0.0.0",
					},
					&cli.IntFlag{
						Name:        "ap",
						Usage:       "agent listener `PORT`",
						Destination: &agentPort,
						Value:       5522,
					},
					&cli.StringFlag{
						Name:        "ah",
						Usage:       "agent listener `HOST`",
						Destination: &agentHost,
						Value:       "0.0.0.0",
					},
				},
				Action: run,
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "db",
				Usage:       "database `PATH`",
				Destination: &dbPath,
				Value:       "rscc.db",
			},
			&cli.BoolFlag{
				Name:        "debug",
				Usage:       "enable debug mode",
				Destination: &debug,
				Value:       false,
			},
		},
		Before: preRun,
	}

	// Run CLI app
	if err := app.Run(ctx, os.Args); err != nil {
		os.Exit(1)
	}
}

func preRun(ctx context.Context, c *cli.Command) (context.Context, error) {
	if debug {
		logger.SetDebug()
	}
	return ctx, nil
}

func addAdmin(ctx context.Context, c *cli.Command) error {
	lg := logger.FromContext(ctx)

	db, err := database.NewDatabase(ctx, dbPath)
	if err != nil {
		lg.Errorf("Failed to initialize database: %v", err)
		return err
	}

	user, err := db.CreateUser(ctx, operatorUsername, publicKey, true)
	if err != nil {
		lg.Errorf("Failed to add operator: %v", err)
		return err
	}

	lg.Infof("New admin `%s` with id `%s` created", user.Name, user.ID)
	return nil
}

func run(ctx context.Context, c *cli.Command) error {
	lg := logger.FromContext(ctx)

	// Initialize database
	db, err := database.NewDatabase(ctx, dbPath)
	if err != nil {
		lg.Errorf("Failed to initialize database: %v", err)
		return err
	}

	// Check if at least one operator exists
	users, err := db.GetAllUsers(ctx)
	if err != nil {
		lg.Errorf("Failed to get all users: %v", err)
		return err
	}
	if len(users) == 0 {
		lg.Errorf("Admin user not found. Use `rscc admin` to create new admin user.")
		return fmt.Errorf("admin user not found")
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
