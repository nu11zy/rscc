package opsrv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"rscc/internal/common/constants"
	"rscc/internal/common/logger"
	"rscc/internal/common/network"
	"rscc/internal/common/pprint"
	"rscc/internal/database"
	"rscc/internal/database/ent"
	"rscc/internal/opsrv/cmd/agentcmd"
	"rscc/internal/opsrv/cmd/operatorcmd"
	"rscc/internal/opsrv/cmd/sessioncmd"
	"rscc/internal/session"
	"rscc/internal/sshd"
	"strings"
	"sync"
	"time"

	"github.com/google/shlex"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type OperatorServer struct {
	db          *database.Database
	sm          *session.SessionManager
	address     string
	listener    *net.TCPListener
	sshConfig   *ssh.ServerConfig
	sshTimeout  int
	operatorsMu sync.Mutex
	lg          *zap.SugaredLogger
}

func NewOperatorServer(ctx context.Context, db *database.Database, sm *session.SessionManager, host string, port int) (*OperatorServer, error) {
	lg := logger.FromContext(ctx).Named("opsrv")
	address := fmt.Sprintf("%s:%d", host, port)

	listener, err := db.GetListener(ctx, constants.OperatorListenerID)
	if err != nil {
		if ent.IsNotFound(err) {
			lg.Info("Operator listener not found, creating new one")
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

	sshTimeout := constants.SSHTimeout
	if sshTimeout < 10 {
		lg.Warnf("SSH timeout is less than 10 seconds, setting to 10 seconds")
		sshTimeout = 10
	}

	opsrv := &OperatorServer{
		db:         db,
		sm:         sm,
		address:    address,
		listener:   nil,
		sshConfig:  nil,
		sshTimeout: sshTimeout,
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
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer listener.Close()

	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		return fmt.Errorf("listener is not *net.TCPListener")
	}
	s.listener = tcpListener
	s.lg.Infof("Operator listener started at %s", s.address)

	go func() {
		<-ctx.Done()
		if err := s.CloseListener(); err != nil {
			s.lg.Errorf("Failed to close listener: %v", err)
		}
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
			if errors.Is(err, net.ErrClosed) {
				s.lg.Warn("Operator listener closed")
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
func (s *OperatorServer) publicKeyCallback(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := s.db.GetOperatorByName(ctx, conn.User())
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	marshaledKey := string(ssh.MarshalAuthorizedKey(key))
	if user.PublicKey != strings.TrimSpace(marshaledKey) {
		s.lg.Warnf("User %s (%s) tried to connect with invalid key", conn.User(), conn.RemoteAddr())
		return nil, fmt.Errorf("invalid key")
	}

	return &ssh.Permissions{}, nil
}

// handleConnection handles new SSH connection
func (s *OperatorServer) handleConnection(conn net.Conn) {
	s.lg.Debugf("New TCP connection from %s", conn.RemoteAddr())
	lg := s.lg.Named(fmt.Sprintf("(%s)", conn.RemoteAddr().String()))

	// create connection with timeout
	timeoutConn := network.NewTimeoutConn(conn, time.Duration(2*s.sshTimeout)*time.Second)

	// create new SSH connection
	sshConn, chans, reqs, err := ssh.NewServerConn(timeoutConn, s.sshConfig)
	if err != nil {
		lg.Errorf("SSH handshake failed: %v", err)
		return
	}
	defer sshConn.Close()

	// start keepalive process
	stopKeepalive := make(chan struct{})
	go func() {
		lg.Debug("Starting keepalive process")
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
			case <-stopKeepalive:
				lg.Debug("Stop sending keepalive requests")
				return
			}
		}
	}()

	lg.Infof("New SSH connection (%s)", sshConn.User())
	lg.Debugf("SSH client version: %s", sshConn.ClientVersion())

	operatorSession := &sshd.OperatorSession{
		Username:    sshConn.User(),
		Permissions: sshConn.Permissions,
	}

	go ssh.DiscardRequests(reqs)
	s.handleChannels(chans, operatorSession)

	// stop keepalive process
	stopKeepalive <- struct{}{}

	lg.Infof("SSH connection closed (%s)", sshConn.User())
}

// handleChannels handles SSH channels
func (s *OperatorServer) handleChannels(chans <-chan ssh.NewChannel, operatorSession *sshd.OperatorSession) {
	lg := s.lg.Named("ssh")

	for newChannel := range chans {
		lg.Debugf("Requested channel: %s", newChannel.ChannelType())
		switch newChannel.ChannelType() {
		case "session":
			rawChannel, request, err := newChannel.Accept()
			if err != nil {
				lg.Errorf("Failed to accept channel: %v", err)
				continue
			}
			channel := sshd.NewExtendedChannel(rawChannel, operatorSession)
			go s.handleSession(channel, request)
		case "direct-tcpip":
			extraData := newChannel.ExtraData()
			channel, _, err := newChannel.Accept()
			if err != nil {
				lg.Errorf("Failed to accept channel: %v", err)
				continue
			}
			go s.handleJump(channel, extraData)
		default:
			lg.Warnf("Unsupported channel type: %s", newChannel.ChannelType())
			newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
		}
	}
}

// handleJump handles connection from operator to agent
func (s *OperatorServer) handleJump(channel ssh.Channel, extraData []byte) {
	lg := s.lg.Named("jmp")
	defer channel.Close()

	connData, err := sshd.GetExtraData(extraData)
	if err != nil {
		lg.Errorf("Failed to get extra data: %v", err)
		return
	}
	lg.Infof("Reverse SSH connection from %s:%d to %s:%d", connData.OriginatorIP, connData.OriginatorPort, connData.TargetHost, connData.TargetPort)

	agentId := strings.Split(string(connData.TargetHost), ":")[1]
	session, ok := s.sm.GetSession(agentId)
	if !ok {
		lg.Warnf("Session not found: %s", agentId)
		return
	}
	lg.Infof("Session found: %s", agentId)

	// TODO: use ssh-jump channel
	sessionConn, sessionReqs, err := session.SSHConn.Conn.OpenChannel("jumphost", nil)
	if err != nil {
		lg.Errorf("Failed to open ssh-jump channel: %v", err)
		return
	}
	defer sessionConn.Close()

	go ssh.DiscardRequests(sessionReqs)
	go func() {
		io.Copy(channel, sessionConn)
	}()
	io.Copy(sessionConn, channel)
}

// handleSession handles SSH session channel
func (s *OperatorServer) handleSession(channel *sshd.ExtendedChannel, request <-chan *ssh.Request) {
	lg := s.lg.Named("ssh")

	isPty := false
	for req := range request {
		lg.Debugf("Session request: %s", req.Type)
		switch req.Type {
		case "pty-req":
			isPty = true
			req.Reply(true, nil)
		case "window-change":
			req.Reply(true, nil)
		case "shell":
			if isPty {
				req.Reply(true, nil)
				go s.handleShell(channel)
			} else {
				lg.Warn("Shell request received before PTY request")
				channel.Write([]byte("Only PTY is supported.\n"))
				req.Reply(true, nil)
				channel.CloseWithStatus(1)
			}
		case "exec":
			go s.handleExec(channel, string(req.Payload[4:]))
			req.Reply(true, nil)
		default:
			lg.Warnf("Unsupported session request: %s", req.Type)
			req.Reply(false, nil)
		}
	}
}

// handleExec handles exec request
func (s *OperatorServer) handleExec(channel *sshd.ExtendedChannel, command string) {
	defer channel.CloseWithStatus(0)

	lg := s.lg.Named("exec")
	lg.Debugf("Executing command: %s", command)

	terminal := term.NewTerminal(channel, "")
	app := s.newCli(terminal, channel.Operator)
	app.SetArgs(strings.Fields(command))

	if err := app.Execute(); err != nil {
		channel.CloseWithStatus(1)
	}
}

// handleShell handles shell request
func (s *OperatorServer) handleShell(channel *sshd.ExtendedChannel) {
	defer channel.CloseWithStatus(0)

	lg := s.lg.Named("cli")
	lg.Infof("Starting CLI for %s", channel.Operator.Username)

	terminal := term.NewTerminal(channel, "rscc > ")
	terminal.Write([]byte(pprint.GetBanner()))
	// TODO: print status (number of sessions, uptime, etc.)

	for {
		cli := s.newCli(terminal, channel.Operator)

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
			cli.PrintErr(fmt.Sprintf("%s Error: %s\n", pprint.ErrorPrefix, err.Error()))
			continue
		}
		cli.SetArgs(args)
		if err := cli.Execute(); err != nil {
			cli.PrintErr(fmt.Sprintf("%s Error: %s\n", pprint.ErrorPrefix, err.Error()))
		}
	}
}

// newCli creates new CLI instance for operator
func (s *OperatorServer) newCli(terminal *term.Terminal, operatorSession *sshd.OperatorSession) *cobra.Command {
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

	app.SetUsageTemplate(`{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "[command] --help" for more information about a command.{{end}}
`)

	app.SetOut(terminal)
	app.SetErr(terminal)

	app.AddCommand(sessioncmd.NewSessionCmd(s.sm).Command)
	app.AddCommand(agentcmd.NewAgentCmd(s.db).Command)
	app.AddCommand(operatorcmd.NewOperatorCmd(s.db, operatorSession).Command)
	return app
}
