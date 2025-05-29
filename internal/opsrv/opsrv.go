package opsrv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"rscc/internal/common/constants"
	"rscc/internal/common/logger"
	"rscc/internal/common/network"
	"rscc/internal/common/pprint"
	"rscc/internal/common/utils"
	"rscc/internal/database"
	"rscc/internal/database/ent"
	"rscc/internal/opsrv/cmd/agentcmd"
	"rscc/internal/opsrv/cmd/sessioncmd"
	"rscc/internal/session"
	"rscc/internal/sshd"
	"strconv"
	"strings"
	"time"

	"github.com/google/shlex"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// OperatorServerConfig holds related to operator's server settings
type OperatorServerConfig struct {
	Host     string
	Port     int
	BasePath string
}

type OperatorServer struct {
	db         *database.Database
	sm         *session.SessionManager
	listener   *net.TCPListener
	sshConfig  *ssh.ServerConfig
	sshTimeout int
	lg         *zap.SugaredLogger
	config     *OperatorServerConfig
}

// NewServer returns prepared object of operator's server
func NewServer(ctx context.Context, db *database.Database, sm *session.SessionManager, config *OperatorServerConfig) (*OperatorServer, error) {
	lg := logger.FromContext(ctx).Named("opsrv")

	// get keys for listener
	listener, err := db.GetListener(ctx, constants.OperatorListenerID)
	if err != nil {
		if ent.IsNotFound(err) {
			lg.Info("Listener not found, creating new one")
			keyPair, err := sshd.NewECDSAKey()
			if err != nil {
				return nil, fmt.Errorf("failed to generate key pair: %w", err)
			}
			privateKey, err := keyPair.GetPrivateKey()
			if err != nil {
				return nil, fmt.Errorf("failed to get private key: %w", err)
			}
			listener, err = db.CreateListenerWithID(
				ctx,
				constants.OperatorListenerID,
				constants.OperatorListenerName,
				privateKey,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create operator listener: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get operator listener: %w", err)
		}
	}

	signer, err := ssh.ParsePrivateKey(listener.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	opsrv := &OperatorServer{
		db:         db,
		sm:         sm,
		config:     config,
		listener:   nil,
		sshConfig:  nil,
		sshTimeout: constants.SshTimeout,
		lg:         lg,
	}

	sshConfig := &ssh.ServerConfig{
		NoClientAuth:      false,
		PublicKeyCallback: opsrv.publicKeyCallback,
	}
	sshConfig.AddHostKey(signer)

	opsrv.sshConfig = sshConfig
	return opsrv, nil
}

// Start starts operator's listener
func (s *OperatorServer) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port)))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer listener.Close()

	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		return fmt.Errorf("listener is not *net.TCPListener")
	}
	s.listener = tcpListener
	s.lg.Infof("Listener started at %s", net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port)))

	go func() {
		<-ctx.Done()
		if err := s.CloseListener(); err != nil {
			s.lg.Errorf("Failed to close listener: %v", err)
		}
		s.lg.Info("Stop listener")
	}()

	for {
		if err := s.listener.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			return fmt.Errorf("failed to set listener deadline: %w", err)
		}

		conn, err := s.listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return nil
			}
			s.lg.Errorf("Failed to accept connection: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

// CloseListener closes operator's listener if it's active
func (l *OperatorServer) CloseListener() error {
	if l.listener != nil {
		return l.listener.Close()
	}
	return nil
}

// publicKeyCallback is used to authenticate SSH connections
func (s *OperatorServer) publicKeyCallback(conn ssh.ConnMetadata, incomingKey ssh.PublicKey) (*ssh.Permissions, error) {
	var authorizedKeys []byte

	// prpare paths for authorized_keys files
	rsccAuthorizedKeysPath := filepath.Join(s.config.BasePath, "authorized_keys")
	globalAuthorizedKeysPath := filepath.Join(os.Getenv("HOME"), ".ssh", "authorized_keys")

	if _, err := os.Stat(rsccAuthorizedKeysPath); err == nil {
		// read rscc authorized_keys in data directory
		authorizedKeys, err = os.ReadFile(rsccAuthorizedKeysPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", rsccAuthorizedKeysPath, err)
		}
	} else if _, err := os.Stat(globalAuthorizedKeysPath); err == nil {
		// read authorized_keys from ~/.ssh/
		authorizedKeys, err = os.ReadFile(globalAuthorizedKeysPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", globalAuthorizedKeysPath, err)
		}
	} else {
		return nil, fmt.Errorf("authorized_keys file not found")
	}

	// Parse authorized_keys
	for len(authorizedKeys) > 0 {
		storedKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeys)
		if err != nil {
			return nil, fmt.Errorf("failed to parse authorized_keys: %w", err)
		}
		authorizedKeys = rest

		if bytes.Equal(storedKey.Marshal(), incomingKey.Marshal()) {
			s.lg.Infof("User %s (%s) successfully authenticated", conn.User(), conn.RemoteAddr())
			return &ssh.Permissions{}, nil
		}
	}

	s.lg.Warnf("User %s (%s) tried to connect with invalid key", conn.User(), conn.RemoteAddr())
	return nil, fmt.Errorf("invalid key")
}

// handleConnection handles new SSH connection
func (s *OperatorServer) handleConnection(conn net.Conn) {
	lg := s.lg

	lg.Debugf("New TCP connection from %s", conn.RemoteAddr().String())

	// create connection with timeout
	timeoutConn := network.NewTimeoutConn(conn, time.Duration(2*s.sshTimeout)*time.Second)
	defer timeoutConn.Close()

	// create new SSH connection
	sshConn, chans, reqs, err := ssh.NewServerConn(timeoutConn, s.sshConfig)
	if err != nil {
		if strings.Contains(err.Error(), "authorized_keys file not found") {
			lg.Errorf("SSH handshake failed: authorized_keys file not found. Please create one in the %s directory or int ~/.ssh/", s.config.BasePath)
			return
		}
		lg.Errorf("SSH handshake failed: %v", err)
		return
	}
	defer sshConn.Close()

	// update logger
	lg = lg.Named(fmt.Sprintf("[%s]", sshConn.User()))

	// start keepalive process
	stopKeepalive := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Duration(s.sshTimeout) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if _, _, err := sshConn.SendRequest("keepalive@openssh.com", false, nil); err != nil {
					lg.Warnf("Failed to send keepalive, assuming SSH client disconnected: %v", err)
					sshConn.Close()
					return
				}
				lg.Debug("Send keepalive request")
			case <-stopKeepalive:
				lg.Debug("Stop sending keepalive requests")
				return
			}
		}
	}()

	lg.Infof("New SSH connection from %s (%s)", sshConn.RemoteAddr().String(), sshConn.ClientVersion())
	go ssh.DiscardRequests(reqs)
	s.handleChannels(lg, chans)

	// stop keepalive process
	stopKeepalive <- struct{}{}

	lg.Infof("SSH connection closed from %s", sshConn.RemoteAddr())
}

// handleChannels handles SSH channels
func (s *OperatorServer) handleChannels(lg *zap.SugaredLogger, channels <-chan ssh.NewChannel) {
	for newChannel := range channels {
		lg.Debugf("Requested channel: %s", newChannel.ChannelType())
		switch newChannel.ChannelType() {
		case "session":
			subLg := lg.Named("session")
			rawChannel, request, err := newChannel.Accept()
			if err != nil {
				lg.Errorf("Failed to accept channel: %v", err)
				continue
			}
			channel := sshd.NewExtendedChannel(rawChannel)
			go s.handleSession(subLg, channel, request)
		case "direct-tcpip":
			go s.handleJump(lg.Named("direct-tcpip"), newChannel)
		default:
			lg.Warnf("Unsupported channel type: %s", newChannel.ChannelType())
			newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
		}
	}
}

// handleJump handles connection from operator to agent
func (s *OperatorServer) handleJump(lg *zap.SugaredLogger, newChannel ssh.NewChannel) {
	// extract extra data to determine proxyjump target
	connData, err := sshd.GetExtraData(newChannel.ExtraData())
	if err != nil {
		lg.Errorf("Failed to get extra data: %v", err)
		return
	}
	lg.Debugf("Reverse SSH connection from %s:%d to %s:%d", connData.OriginatorIP, connData.OriginatorPort, connData.TargetHost, connData.TargetPort)

	// get name of agent (partial)
	splittedHost := strings.Split(string(connData.TargetHost), "+")
	if len(splittedHost) != 2 {
		lg.Warnf("Invalid format of host %s", connData.TargetHost)
		newChannel.Reject(ssh.Prohibited, fmt.Sprintf("\n\nInvalid format for proxyjump '%s'\n", connData.TargetHost))
		return
	}
	agentId := splittedHost[1]
	session := s.sm.GetSession(agentId)
	if session == nil {
		lg.Warnf("Session not found for proxyjump: %s", agentId)
		newChannel.Reject(ssh.Prohibited, fmt.Sprintf("\n\nNo clients matched '%s'\n", agentId))
		return
	}
	lg.Debugf("Session found for proxyjump: %s", session.ID)

	// update logger
	lg = lg.Named(fmt.Sprintf("[%s]", session.ID))

	// custom ssh-jump SSH channel
	sessionConn, sessionReqs, err := session.SSHConn.Conn.OpenChannel("ssh-jump", nil)
	if err != nil {
		lg.Errorf("Failed to open ssh-jump channel for proxyjump: %v", err)
		newChannel.Reject(ssh.ConnectionFailed, fmt.Sprintf("\n\n%v\n", err.Error()))
		return
	}
	lg.Info("Open ssh-jump channel for proxyjump")
	defer sessionConn.Close()
	go ssh.DiscardRequests(sessionReqs)

	// accept channel to process data forwarding
	channel, channelRequests, err := newChannel.Accept()
	if err != nil {
		newChannel.Reject(ssh.ConnectionFailed, fmt.Sprintf("\n\n%v\n", err.Error()))
		return
	}
	defer channel.Close()
	go ssh.DiscardRequests(channelRequests)

	// process data forwarding
	go func() {
		io.Copy(channel, sessionConn)
		channel.Close()
	}()
	io.Copy(sessionConn, channel)
}

// handleSession handles SSH session channel
func (s *OperatorServer) handleSession(lg *zap.SugaredLogger, channel *sshd.ExtendedChannel, request <-chan *ssh.Request) {
	isPty := false
	terminal := term.NewTerminal(channel, "")
	for req := range request {
		lg.Debugf("Session request: %s", req.Type)
		switch req.Type {
		case "pty-req":
			subLg := lg.Named("pty-req")
			isPty = true
			p, err := sshd.ParsePtyReq(req)
			if err != nil {
				subLg.Errorf("Failed to parse pty request: %v", err)
				req.Reply(false, nil)
				continue
			}
			subLg.Infof("%s %dx%d", p.Term, p.Columns, p.Rows)
			terminal.SetSize(int(p.Columns), int(p.Rows))
			req.Reply(true, nil)
		case "window-change":
			subLg := lg.Named("window-changed")
			if len(req.Payload) < 8 {
				subLg.Warn("window-change request received with malformed payload (<8 bytes)")
				req.Reply(true, nil)
				continue
			}
			columns, rows := sshd.ParseWindowChangeReq(req.Payload)
			subLg.Infof("%dx%d", columns, rows)
			terminal.SetSize(int(columns), int(rows))
			req.Reply(true, nil)
		case "shell":
			subLg := lg.Named("shell")
			if isPty {
				go s.handleShell(subLg, channel, terminal)
				req.Reply(true, nil)
			} else {
				subLg.Warn("Shell request received before PTY request")
				channel.Write([]byte("Only PTY is supported.\n"))
				req.Reply(true, nil)
				channel.CloseWithStatus(1)
			}
		case "exec":
			subLg := lg.Named("exec")
			terminal = term.NewTerminal(channel, "")
			go s.handleExec(subLg, channel, terminal, string(req.Payload[4:]))
			req.Reply(true, nil)
		case "subsystem":
			subLg := lg.Named("subsystem")
			system := string(req.Payload[4:])
			subLg.Debugf("Subsystem request received: %s", system)

			if system == "sftp" {
				go sftpHandler(subLg, channel, s.config.BasePath)
				req.Reply(true, nil)
			} else {
				subLg.Warnf("Subsystem not supported: %s", system)
				req.Reply(false, nil)
			}
		default:
			lg.Warnf("Unsupported session request: %s", req.Type)
			req.Reply(false, nil)
		}
	}
}

// handleExec handles exec request
func (s *OperatorServer) handleExec(lg *zap.SugaredLogger, channel *sshd.ExtendedChannel, terminal *term.Terminal, command string) {
	defer channel.CloseWithStatus(0)

	lg.Debugf("Executing command: %s", command)

	app := s.newCli(terminal)
	app.SetArgs(strings.Fields(command))

	if err := app.Execute(); err != nil {
		channel.CloseWithStatus(1)
	}
}

// handleShell handles shell request
func (s *OperatorServer) handleShell(lg *zap.SugaredLogger, channel *sshd.ExtendedChannel, terminal *term.Terminal) {
	defer channel.CloseWithStatus(0)

	lg.Info("Starting rscc CLI")

	terminal.SetPrompt(fmt.Sprintf("%s > ", pprint.Green.Sprint("rscc")))
	terminal.Write([]byte(pprint.GetBanner()))

	for {
		cli := s.newCli(terminal)

		line, err := terminal.ReadLine()
		if err != nil {
			if err == io.EOF {
				return
			}
			lg.Errorf("Failed to read line: %v", err)
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if line == "exit" {
			return
		}

		args, err := shlex.Split(line)
		if err != nil {
			cli.PrintErr(fmt.Sprintf("%s Error: %s\n\n", pprint.ErrorPrefix, err.Error()))
			continue
		}
		cli.SetArgs(args)
		if err := cli.Execute(); err != nil {
			cli.PrintErr(fmt.Sprintf("%s Error: %s\n\n", pprint.ErrorPrefix, err.Error()))
		}
	}
}

// newCli creates new CLI instance for operator
func (s *OperatorServer) newCli(terminal *term.Terminal) *cobra.Command {
	app := &cobra.Command{
		Use:                "rscc",
		Short:              "Reverse SSH command & control",
		DisableFlagParsing: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}
	app.Flags().BoolP("help", "h", false, "")
	app.Flags().MarkHidden("help")
	app.PersistentFlags().BoolP("help", "h", false, "Print this help message")
	app.PersistentFlags().MarkHidden("help")
	app.SetHelpCommand(&cobra.Command{
		Use:    "help",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().Help()
		},
	})

	app.SetUsageFunc(utils.CobraHelp)

	app.SetOut(terminal)
	app.SetErr(terminal)

	app.AddCommand(sessioncmd.NewSessionCmd(s.sm).Command)
	app.AddCommand(agentcmd.NewAgentCmd(s.db, s.config.BasePath).Command)
	return app
}
