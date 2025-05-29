package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"rscc/internal/common/constants"
	"rscc/internal/common/logger"
	"rscc/internal/common/utils"
	"rscc/internal/database"
	"rscc/internal/multiplexer"
	"rscc/internal/multiplexer/ssh"
	"rscc/internal/opsrv"
	"rscc/internal/session"
	"strconv"
	"time"

	"github.com/go-faster/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
)

type Cmd struct {
	OperatorHost    string
	OperatorPort    uint16
	MultiplexerHost string
	MultiplexerPort uint16
	DataDirectory   string
	Timeout         uint
	Debug           bool
}

// RegisterFlags registers flags based on structure
func (c *Cmd) RegisterFlags(f *pflag.FlagSet) error {
	// settings for operator
	f.StringVar(&c.OperatorHost, "operator-host", "0.0.0.0", "operator host to listen on")
	f.Uint16Var(&c.OperatorPort, "operator-port", 55022, "operator port to listen on")

	// settings for multiplexer
	f.StringVar(&c.MultiplexerHost, "multiplexer-host", "0.0.0.0", "multiplexer host to listen on")
	f.Uint16Var(&c.MultiplexerPort, "multiplexer-port", 8080, "multiplexer port to listen on")

	// settings for timeout
	f.UintVarP(&c.Timeout, "timeout", "t", 15, "timeout when client considered as dead")

	// debug logging
	f.BoolVarP(&c.Debug, "debug", "d", false, "enable debug logging")

	// prepare data directory pathpath
	if cwd, err := os.Getwd(); err != nil {
		return err
	} else {
		f.StringVarP(&c.DataDirectory, "data-directory", "D", filepath.Join(cwd, "data"), "data directory")
	}

	return nil
}

// ValidateFlags validates flags
func (c *Cmd) ValidateFlags(ctx context.Context) error {
	var err error
	lg := logger.FromContext(ctx).Named("cmd")

	// validate data directory
	if c.DataDirectory == "" {
		return fmt.Errorf("specify data directory")
	}
	if c.DataDirectory, err = filepath.Abs(c.DataDirectory); err != nil {
		return errors.Wrap(err, "couldn't resolve data directory path")
	}
	if isDir, err := utils.IsDir(c.DataDirectory); err != nil {
		// create data directory if not exists yet
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(c.DataDirectory, os.ModePerm); err != nil {
				return errors.Wrapf(err, "unable create data directory %s", c.DataDirectory)
			}
			lg.Infof("Create data directory %s", c.DataDirectory)
		} else {
			return errors.Wrapf(err, "unable get information about %s", c.DataDirectory)
		}
	} else {
		if !isDir {
			return fmt.Errorf("%s is not valid directory", c.DataDirectory)
		}
	}

	// validate nested directories in data directory
	agentDir := filepath.Join(c.DataDirectory, constants.AgentDir)
	if isDir, err := utils.IsDir(agentDir); err != nil {
		// create agents directory if not exists yet
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(agentDir, os.ModePerm); err != nil {
				return errors.Wrapf(err, "unable create data directory %s", agentDir)
			}
			lg.Infof("Create data directory %s", agentDir)
		} else {
			return errors.Wrapf(err, "unable get information about %s", agentDir)
		}
	} else {
		if !isDir {
			return fmt.Errorf("%s is not valid directory", agentDir)
		}
	}

	// validate timeout
	if c.Timeout == 0 || c.Timeout > constants.MaxClientTimeout {
		return fmt.Errorf("specify timeout in range 1..%d", constants.MaxClientTimeout-1)
	}

	// validate operator's host
	if c.OperatorHost == "" {
		return fmt.Errorf("specify operator's host")
	} else {
		if _, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(c.OperatorHost, strconv.Itoa(int(c.OperatorPort)))); err != nil {
			return errors.Wrap(err, "invalid operator's address")
		}
	}

	// validate multiplexer's host
	if c.MultiplexerHost == "" {
		return fmt.Errorf("specify multiplexer's host")
	} else {
		if _, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(c.MultiplexerHost, strconv.Itoa(int(c.MultiplexerPort)))); err != nil {
			return errors.Wrap(err, "invalid multiplexer's address")
		}
	}
	return nil
}

// Pre prepares environment befor main execution
func (c *Cmd) Pre(cmd *cobra.Command, args []string) error {
	if c.Debug {
		logger.SetDebug()
	}
	return c.ValidateFlags(cmd.Context())
}

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
	opsrv, err := opsrv.NewServer(ctx, db, sm, operatorData)
	if err != nil {
		return errors.Wrap(err, "failed to initialize operator's server")
	}

	// multiplexer listener
	multiplexerData := &multiplexer.MultiplexerConfig{
		Host:           c.MultiplexerHost,
		Port:           int(c.MultiplexerPort),
		BasePath:       c.DataDirectory,
		IsHttpDownload: false,
		IsTcpDownload:  false,
		IsTls:          false,
		Timeout:        time.Duration(time.Duration(c.Timeout) * time.Second),
	}
	mux, err := multiplexer.NewServer(ctx, multiplexerData)
	if err != nil {
		return errors.Wrap(err, "failed to initialize multiplexer's server")
	}

	// ssh agent listener
	if mux.GetSshListener() == nil {
		return fmt.Errorf("no suitable listener found for agent SSH server")
	}
	sshConfig := &ssh.SshConfig{
		Listener: mux.GetSshListener(),
		Timeout:  time.Duration(time.Duration(c.Timeout) * time.Second),
	}
	ssh, err := ssh.NewListener(ctx, db, sm, sshConfig)
	if err != nil {
		return errors.Wrap(err, "failed to initialize agent's SSH server")
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return opsrv.Start(ctx) })
	g.Go(func() error { return mux.Start(ctx) })
	g.Go(func() error { return ssh.Start(ctx) })
	return g.Wait()
}
