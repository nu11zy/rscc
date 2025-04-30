// Reverse SSH agent (starts without arguments and connects to the server)
package main

import (
	"agent/internal/metadata"
	"agent/internal/sshd"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"

	"github.com/creack/pty"
	"golang.org/x/crypto/ssh"
)

var serverAddress = "127.0.0.1:5522"
var sshClientVersion = "SSH-2.0-OpenSSH_9.9"

// SRV <-> TCP <-> SSH_CHAN <-> SRV_PIPE <-> AGENT_PIPE <-> AGENT_SSH_SRV <-> SSH_CHAN <-> PTY
func main() {
	log.Println("Starting agent")

	metadata, err := metadata.GetMetadata()
	if err != nil {
		log.Printf("Failed to get metadata: %v", err)
		return
	}

	sshConfig := &ssh.ClientConfig{
		User:            metadata,
		Auth:            []ssh.AuthMethod{}, // TODO: Add authentication with key
		ClientVersion:   sshClientVersion,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Check host key
	}

	// 1. TCP connection to server
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		log.Printf("Failed to connect to server: %v", err)
		return
	}
	defer conn.Close()

	// 2. SSH handshake
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, serverAddress, sshConfig)
	if err != nil {
		log.Printf("Failed to create SSH connection: %v", err)
		return
	}
	defer sshConn.Close()

	// 3. Handle requests
	go ssh.DiscardRequests(reqs)

	// 4. Handle channels
	for newChannel := range chans {
		log.Printf("New channel: %v", newChannel.ChannelType())
		switch newChannel.ChannelType() {
		case "jumphost":
			channel, request, err := newChannel.Accept()
			if err != nil {
				log.Printf("Failed to accept channel: %v", err)
				newChannel.Reject(ssh.ConnectionFailed, "Failed to accept channel")
				continue
			}
			go handleSession(channel, request)
		default:
			log.Printf("Unknown channel type: %v", newChannel.ChannelType())
			newChannel.Reject(ssh.UnknownChannelType, "Unknown channel type")
		}
	}
}

func handleSession(channel ssh.Channel, request <-chan *ssh.Request) {
	defer channel.Close()
	log.Printf("Session channel accepted")

	sshConfig := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	privKey, err := sshd.GeneratePrivateKey()
	if err != nil {
		log.Printf("Failed to generate private key: %v", err)
		return
	}
	signer, err := ssh.ParsePrivateKey(privKey)
	if err != nil {
		log.Printf("Failed to create signer: %v", err)
		return
	}
	sshConfig.AddHostKey(signer)

	pAgent, pServer := net.Pipe()
	defer pAgent.Close()
	defer pServer.Close()

	go func() {
		_, err := io.Copy(channel, pServer)
		if err != nil {
			log.Printf("io channel<-pServer error: %v", err)
		}
		channel.Close()
	}()
	go func() {
		_, err := io.Copy(pServer, channel)
		if err != nil {
			log.Printf("io pServer<-channel error: %v", err)
		}
		pServer.Close()
	}()

	sshConn, chans, reqs, err := ssh.NewServerConn(pAgent, sshConfig)
	if err != nil {
		log.Printf("Failed to create SSH connection: %v", err)
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		log.Printf("New channel: %v", newChannel.ChannelType())
		switch newChannel.ChannelType() {
		case "session":
			channel, request, err := newChannel.Accept()
			if err != nil {
				log.Printf("Failed to accept channel: %v", err)
				newChannel.Reject(ssh.ConnectionFailed, "Failed to accept channel")
				continue
			}
			go handleJumpSession(channel, request)
		default:
			log.Printf("Unknown channel type: %v", newChannel.ChannelType())
			newChannel.Reject(ssh.UnknownChannelType, "Unknown channel type")
		}
	}
}

func handleJumpSession(channel ssh.Channel, request <-chan *ssh.Request) {
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
				log.Printf("Shell request received before PTY request")
				fmt.Fprintf(channel, "Only PTY requests are supported.\n")
				req.Reply(true, nil)
				channel.Close()
			}
		default:
			log.Printf("Unknown request: %v", req.Type)
			req.Reply(false, nil)
		}
	}
}

func handleShell(channel ssh.Channel) {
	defer channel.Close()

	bash := exec.Command("/bin/bash")
	ptyFile, err := pty.Start(bash)
	if err != nil {
		log.Printf("Failed to start bash: %v", err)
		return
	}
	defer ptyFile.Close()

	go io.Copy(ptyFile, channel)
	go io.Copy(channel, ptyFile)

	if err := bash.Wait(); err != nil {
		log.Printf("Bash exited with error: %v", err)
	}
}
