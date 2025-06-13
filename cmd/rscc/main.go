package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"rscc/cmd/rscc/internal/cmd"
	"rscc/internal/common/logger"

	"github.com/spf13/cobra"
)

var (
	OperatorPort uint16
	OperatorHost string
	AgentPort    uint16
	AgentHost    string
	TlsCertPath  string
	TlsKeyPath   string
	HtmlPagePath string
	DataPath     string
	Debug        bool
)

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
	cmd := &cmd.Cmd{}
	root := &cobra.Command{
		Use:     "rscc",
		Short:   "Reverse SSH command & control",
		PreRunE: cmd.PreRunE,
		RunE:    cmd.RunE,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		SilenceUsage: true,
	}
	cmd.RegisterFlags(root.Flags())
	root.MarkFlagsRequiredTogether("tls-cert", "tls-key")

	if err := root.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
