//go:build !windows
// +build !windows

package sshd

import (
	"agent/internal/logger"
	"errors"
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"golang.org/x/crypto/ssh"
)

type unixShell struct {
	ptyFile *os.File
	shell   *exec.Cmd
	columns uint32
	rows    uint32
}

func (s *unixShell) SetSize(req *ssh.Request) error {
	columns, rows, err := ParseSize(req)
	if err != nil {
		return err
	}
	s.columns = columns
	s.rows = rows

	if s.ptyFile != nil {
		err := pty.Setsize(s.ptyFile, &pty.Winsize{
			Rows: uint16(rows),
			Cols: uint16(columns),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *unixShell) HandleShell(channel ssh.Channel) {
	defer channel.Close()
	defer s.ptyFile.Close()
	defer s.shell.Process.Kill()

	go io.Copy(s.ptyFile, channel)
	go io.Copy(channel, s.ptyFile)

	lg := logger.GetLogger()
	if err := s.shell.Wait(); err != nil {
		lg.Error("Shell exited with error: %v", err)
	}
}

func NewShell() (sshShell, error) {
	var shell *exec.Cmd
	if _, err := os.Stat("/bin/bash"); err == nil { // TODO: check shells in /etc/shells
		shell = exec.Command("/bin/bash", "--noprofile", "--norc")
	} else {
		shell = exec.Command("/bin/sh")
	}

	if shell == nil {
		return nil, errors.New("shell binary not found")
	}
	shell.Env = os.Environ()
	shell.Env = append(shell.Env, "HISTFILE=")

	ptyFile, err := pty.Start(shell)
	if err != nil {
		return nil, err
	}

	return &unixShell{
		ptyFile: ptyFile,
		shell:   shell,
		columns: 80,
		rows:    24,
	}, nil
}
