package sshd

import (
	"fmt"
	"io"

	// {{if .Debug}}
	"log"
	// {{end}}
	"net"

	"golang.org/x/crypto/ssh"
)

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

func handleJump(channel ssh.Channel, request <-chan *ssh.Request, sshServerConfig *ssh.ServerConfig) {
	defer channel.Close()

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
	for req := range request {
		switch req.Type {
		case "pty-req":
			isPty = true
			req.Reply(true, nil)
		case "window-change":
			req.Reply(true, nil)
		case "shell":
			if isPty {
				go handleShell(channel)
				req.Reply(true, nil)
			} else {
				// {{if .Debug}}
				log.Printf("Shell request received before PTY request")
				// {{end}}
				fmt.Fprintf(channel, "Only PTY requests are supported.\n")
				req.Reply(true, nil)
				return
			}
		case "subsystem":
			// {{if .Debug}}
			log.Printf("Subsystem request received: %v", req.Payload)
			// {{end}}
			system := string(req.Payload[4:])
			switch system {
			case "sftp":
				// {{if .Debug}}
				log.Printf("SFTP subsystem request received")
				// {{end}}
				req.Reply(true, nil)
			default:
				// {{if .Debug}}
				log.Printf("Unknown subsystem: %v", system)
				// {{end}}
				req.Reply(false, []byte(fmt.Sprintf("Subsystem %s not supported.", system)))
			}
		default:
			// {{if .Debug}}
			log.Printf("Unknown request: %v", req.Type)
			// {{end}}
			req.Reply(false, []byte("Unknown request"))
		}
	}
}
