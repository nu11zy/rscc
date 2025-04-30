package opsrv

import (
	"context"
	"fmt"
	"io"
	"net"
	"rscc/internal/common/logger"
	"rscc/internal/database"
	"rscc/internal/database/ent"
	"rscc/internal/session"
	"rscc/internal/sshd"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type OperatorServer struct {
	db        *database.Database
	sm        *session.SessionManager
	address   string
	sshConfig *ssh.ServerConfig
	lg        *zap.SugaredLogger
}

const (
	OperatorListenerName = "operator"
	OperatorListenerID   = "00000000"
)

func NewOperatorServer(ctx context.Context, db *database.Database, sm *session.SessionManager, host string, port int) (*OperatorServer, error) {
	lg := logger.FromContext(ctx).Named("opsrv")
	address := fmt.Sprintf("%s:%d", host, port)

	listener, err := db.GetListener(ctx, OperatorListenerID)
	if err != nil {
		if ent.IsNotFound(err) {
			lg.Info("Operator listener not found, creating new one")
			privateKey, err := sshd.GeneratePrivateKey()
			if err != nil {
				return nil, fmt.Errorf("failed to generate private key: %w", err)
			}
			listener, err = db.CreateListenerWithID(ctx, OperatorListenerID, OperatorListenerName, privateKey)
			if err != nil {
				return nil, fmt.Errorf("failed to create operator listener: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get operator listener: %w", err)
		}
	}

	sshConfig := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	signer, err := ssh.ParsePrivateKey(listener.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	sshConfig.AddHostKey(signer)

	return &OperatorServer{
		db:        db,
		sm:        sm,
		address:   address,
		sshConfig: sshConfig,
		lg:        lg,
	}, nil
}

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
	s.lg.Infof("Operator server started at %s", s.address)

	for {
		err := tcpListener.SetDeadline(time.Now().Add(2 * time.Second))
		if err != nil {
			return fmt.Errorf("failed to set deadline: %w", err)
		}

		conn, err := tcpListener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				select {
				case <-ctx.Done():
					s.lg.Info("Operator server stopped")
					return nil
				default:
					continue
				}
			}
			s.lg.Errorf("failed to accept connection: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *OperatorServer) handleConnection(conn net.Conn) {
	lg := s.lg.Named("tcp")
	lg.Debugf("New connection from %s", conn.RemoteAddr())

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.sshConfig)
	if err != nil {
		lg.Errorf("SSH handshake failed: %v", err)
		return
	}
	defer sshConn.Close()

	user := sshConn.User()
	lg.Infof("New SSH connection from %s (%s)", sshConn.RemoteAddr(), user)
	lg.Debugf("SSH client version: %s", sshConn.ClientVersion())

	go ssh.DiscardRequests(reqs)
	s.handleChannels(chans)

	lg.Infof("SSH connection closed from %s (%s)", sshConn.RemoteAddr(), user)

}

func (s *OperatorServer) handleChannels(chans <-chan ssh.NewChannel) {
	lg := s.lg.Named("ssh")
	for newChannel := range chans {
		lg.Debugf("Requested channel: %s", newChannel.ChannelType())
		switch newChannel.ChannelType() {
		case "session":
			channel, request, err := newChannel.Accept()
			if err != nil {
				lg.Errorf("Failed to accept channel: %v", err)
				continue
			}
			go s.handleSession(channel, request)
		case "direct-tcpip":
			extraData := newChannel.ExtraData()
			channel, request, err := newChannel.Accept()
			if err != nil {
				lg.Errorf("Failed to accept channel: %v", err)
				continue
			}
			go s.handleReverseSSH(channel, request, extraData)
		default:
			lg.Warnf("Unsupported channel type: %s", newChannel.ChannelType())
			newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
		}
	}
}

type ExtraData struct {
	Host           string
	Port           uint32
	OriginatorIP   string
	OriginatorPort uint32
}

func (s *OperatorServer) handleReverseSSH(channel ssh.Channel, request <-chan *ssh.Request, extraData []byte) {
	lg := s.lg.Named("ssh")

	var connData ExtraData
	err := ssh.Unmarshal(extraData, &connData)
	if err != nil {
		lg.Errorf("Failed to unmarshal extra data: %v", err)
		return
	}
	lg.Infof("Reverse SSH connection to %s:%d from %s:%d", connData.Host, connData.Port, connData.OriginatorIP, connData.OriginatorPort)

	agentId := strings.Split(string(connData.Host), "-")[1]
	session, ok := s.sm.GetSession(agentId)
	if !ok {
		lg.Errorf("Session not found: %s", agentId)
		return
	}
	lg.Infof("Session found: %s", agentId)

	sessionConn, sessionReqs, err := session.SSHConn.Conn.OpenChannel("jumphost", nil)
	if err != nil {
		lg.Errorf("Failed to open jumphost channel: %v", err)
		return
	}
	defer sessionConn.Close()
	go ssh.DiscardRequests(sessionReqs)

	go func() {
		io.Copy(channel, sessionConn)
		channel.Close()
	}()
	io.Copy(sessionConn, channel)
}

func (s *OperatorServer) handleSession(channel ssh.Channel, request <-chan *ssh.Request) {
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
				go s.handleShell(channel)
				req.Reply(true, nil)
			} else {
				lg.Warn("Shell request received before PTY request")
				fmt.Fprintf(channel, "Only PTY requests are supported.\n")
				req.Reply(true, nil)
				channel.Close()
			}
		default:
			lg.Warnf("Unsupported session request: %s", req.Type)
			req.Reply(false, nil)
		}
	}
}

func (s *OperatorServer) handleShell(channel ssh.Channel) {
	defer channel.Close()

	lg := s.lg.Named("cli")
	lg.Debug("Starting CLI")

	terminal := term.NewTerminal(channel, "rscc > ")
	terminal.Write([]byte("Welcome to the operator shell\n"))

	for {
		app := s.newCli(terminal)

		line, err := terminal.ReadLine()
		if err != nil {
			if err == io.EOF {
				lg.Info("EOF received, exiting")
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

		app.SetArgs(strings.Fields(line))
		app.Execute()
	}
}

func (s *OperatorServer) newCli(terminal *term.Terminal) *cobra.Command {
	app := &cobra.Command{
		Use:                "rscc",
		Short:              "reverse SSH command & control",
		DisableFlagParsing: true,
		SilenceUsage:       true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}
	app.Flags().BoolP("help", "h", false, "")
	app.Flags().MarkHidden("help")
	app.SetOut(terminal)
	app.SetErr(terminal)

	app.AddCommand(s.NewSessionCommand())
	app.AddCommand(s.NewAgentCommand())
	return app
}
