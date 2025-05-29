package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	realhttp "net/http"
	"os"
	"path/filepath"
	"rscc/internal/common/constants"
	"rscc/internal/common/logger"
	"rscc/internal/common/utils"
	"strconv"

	"github.com/go-faster/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Cmd struct {
	OperatorHost      string
	OperatorPort      uint16
	MultiplexerHost   string
	MultiplexerPort   uint16
	DataDirectory     string
	Timeout           uint
	Debug             bool
	HttpPlugPagePath  string
	HttpPlugPageCode  int
	HttpPlugPageBytes []byte
	HttpDownload      bool
	TcpDownload       bool
	TlsEnabled        bool
	TlsCertPath       string
	TlsKeyPath        string
	TlsCertificate    tls.Certificate
}

// RegisterFlags registers flags based on structure
func (c *Cmd) RegisterFlags(f *pflag.FlagSet) error {
	// settings for operator
	f.StringVar(&c.OperatorHost, "operator-host", "0.0.0.0", "operator host to listen on")
	f.Uint16Var(&c.OperatorPort, "operator-port", 55022, "operator port to listen on")

	// settings for multiplexer
	f.StringVar(&c.MultiplexerHost, "multiplexer-host", "0.0.0.0", "multiplexer host to listen on")
	f.Uint16Var(&c.MultiplexerPort, "multiplexer-port", 8080, "multiplexer port to listen on")

	// settings for HTTP server
	f.StringVar(&c.HttpPlugPagePath, "plug-page-path", "", "path to custom plug page for HTTP server")
	f.IntVar(&c.HttpPlugPageCode, "plug-page-code", 200, "http code to custom plug page for HTTP server")
	f.BoolVar(&c.HttpDownload, "download-http", false, "enable HTTP delivery of agents")

	// settings for TLS
	f.BoolVar(&c.TlsEnabled, "tls", false, "enable TLS for HTTP server")
	f.StringVar(&c.TlsCertPath, "tls-cert-path", "", "path to TLS certificate")
	f.StringVar(&c.TlsKeyPath, "tls-key-path", "", "path to TLS private key")

	// settings for TCP server
	f.BoolVar(&c.TcpDownload, "download-tcp", false, "enable TCP delivery of agents")

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

	// validate plug page
	if c.HttpPlugPagePath != "" {
		c.HttpPlugPageBytes, err = os.ReadFile(c.HttpPlugPagePath)
		if err != nil {
			return errors.Wrapf(err, "get plug page by path %s", c.HttpPlugPagePath)
		}
		lg.Infof("Use plug page for HTTP server from path %s", c.HttpPlugPagePath)
	} else {
		// default plug page
		c.HttpPlugPageBytes = []byte(constants.BadGatewayPage)
		c.HttpPlugPageCode = 502
	}

	// validate plug page code by RFC
	if realhttp.StatusText(c.HttpPlugPageCode) == "" {
		return fmt.Errorf("specify valid plug page code, not %d", c.HttpPlugPageCode)
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

	// validate tls
	if c.TlsEnabled {
		if c.TlsCertPath == "" && c.TlsKeyPath == "" {
			// TODO: remove harcode
			cert, err := utils.GenTlsCertificate("127.0.0.1")
			if err != nil {
				return errors.Wrap(err, "generate self-signed TLS certificate")
			}
			c.TlsCertificate = cert
		} else {
			// check if both TLS paths set
			if c.TlsCertPath != "" {
				if c.TlsKeyPath == "" {
					return errors.New("specify TLS key path")
				}
				if ok, err := utils.IsFile(c.TlsCertPath); err != nil {
					return errors.Wrap(err, "TLS certificate path")
				} else {
					if !ok {
						return fmt.Errorf("TLS certificate path %s is not valid file", c.TlsCertPath)
					}
				}
			}
			if c.TlsKeyPath != "" {
				if c.TlsCertPath == "" {
					return errors.New("specify TLS certificate path")
				}
				if ok, err := utils.IsFile(c.TlsKeyPath); err != nil {
					return errors.Wrap(err, "TLS key path")
				} else {
					if !ok {
						return fmt.Errorf("TLS key path %s is not valid file", c.TlsCertPath)
					}
				}
			}
			// create TLS certificate chain
			cert, err := tls.LoadX509KeyPair(c.TlsCertPath, c.TlsKeyPath)
			if err != nil {
				return errors.Wrap(err, "create TLS certifiacte chain")
			}
			c.TlsCertificate = cert
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
