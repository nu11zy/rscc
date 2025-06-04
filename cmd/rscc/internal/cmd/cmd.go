package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"rscc/internal/common/logger"
	"rscc/internal/common/validators"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Cmd struct {
	OperatorPort int
	OperatorHost string
	AgentPort    int
	AgentHost    string
	TlsCertPath  string
	TlsKeyPath   string
	HtmlPagePath string
	DataPath     string
	Debug        bool
}

func (c *Cmd) RegisterFlags(fs *pflag.FlagSet) error {
	fs.IntVar(&c.OperatorPort, "op", 55022, "operator listener port")
	fs.StringVar(&c.OperatorHost, "oh", "0.0.0.0", "operator listener host")
	fs.IntVar(&c.AgentPort, "ap", 8080, "agent listener port")
	fs.StringVar(&c.AgentHost, "ah", "0.0.0.0", "agent listener host")
	fs.StringVarP(&c.TlsCertPath, "tls-cert", "c", "", "TLS certificate path")
	fs.StringVarP(&c.TlsKeyPath, "tls-key", "k", "", "TLS key path")
	fs.StringVarP(&c.HtmlPagePath, "page", "p", "", "fake HTML page path")
	fs.StringVarP(&c.DataPath, "data", "d", "", "data directory path")
	fs.BoolVar(&c.Debug, "debug", false, "enable debug logging")

	return nil
}

func (c *Cmd) PreRunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	lg := logger.FromContext(ctx).Named("cmd")

	if c.Debug {
		logger.SetDebug()
	}

	if err := c.ValidateFlags(ctx); err != nil {
		return err
	}

	// Create data directory if it doesn't exist
	if !validators.ValidateDirectoryExists(c.DataPath) {
		if err := os.Mkdir(c.DataPath, 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %v", err)
		}
		lg.Infof("Created data directory: %s", c.DataPath)
	}

	return nil
}

func (c *Cmd) ValidateFlags(ctx context.Context) error {
	// Validate operator port
	if !validators.ValidatePort(c.OperatorPort) {
		return fmt.Errorf("invalid operator port: %d", c.OperatorPort)
	}

	// Validate agent port
	if !validators.ValidatePort(c.AgentPort) {
		return fmt.Errorf("invalid agent port: %d", c.AgentPort)
	}

	// Validate operator host
	if !validators.ValidateHost(c.OperatorHost) {
		return fmt.Errorf("invalid operator host: %s", c.OperatorHost)
	}

	// Validate agent host
	if !validators.ValidateHost(c.AgentHost) {
		return fmt.Errorf("invalid agent host: %s", c.AgentHost)
	}

	// TODO: Check absolute path for all paths
	// Validate tls cert
	if c.TlsCertPath != "" {
		if !validators.ValidateFileExists(c.TlsCertPath) {
			return fmt.Errorf("invalid tls cert path: %s", c.TlsCertPath)
		}
	}

	// Validate tls key
	if c.TlsKeyPath != "" {
		if !validators.ValidateFileExists(c.TlsKeyPath) {
			return fmt.Errorf("invalid tls key path: %s", c.TlsKeyPath)
		}
	}

	// Validate html page path
	if c.HtmlPagePath != "" {
		if !validators.ValidateFileExists(c.HtmlPagePath) {
			return fmt.Errorf("invalid html page path: %s", c.HtmlPagePath)
		}
	}

	// Validate data path
	if c.DataPath != "" {
		absPath, err := filepath.Abs(c.DataPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for data path: %v", err)
		}
		c.DataPath = absPath
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %v", err)
		}
		c.DataPath = filepath.Join(cwd, "data")
	}

	return nil
}
