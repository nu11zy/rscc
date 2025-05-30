package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"rscc/cmd/rscc/internal/cmd"
	"rscc/internal/common/logger"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
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
	app := &cmd.Cmd{}
	root := &cobra.Command{
		Use:     "rscc",
		Short:   "Reverse SSH command & control",
		PreRunE: app.Pre,
		RunE:    app.Run,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// register flags
	if err := app.RegisterFlags(root.PersistentFlags()); err != nil {
		color.Red("%v", err)
		os.Exit(1)
	}
	// execute program
	if err := root.ExecuteContext(ctx); err != nil {
		color.Red("%v", err)
		os.Exit(2)
	}
}
