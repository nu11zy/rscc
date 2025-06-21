//go:build windows
// +build windows

package sshd

import (
	"agent/internal/logger"
	"io"

	pty "github.com/aymanbagabas/go-pty"
	"golang.org/x/crypto/ssh"
)

type winShell struct {
	ptyFile pty.Pty
	columns uint32
	rows    uint32
}

func (s *winShell) SetSize(req *ssh.Request) error {
	columns, rows, err := ParseSize(req)
	if err != nil {
		return err
	}
	s.columns = columns
	s.rows = rows

	if s.ptyFile != nil {
		err := s.ptyFile.Resize(int(columns), int(rows))
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *winShell) HandleShell(channel ssh.Channel) {
	defer channel.Close()
	defer s.ptyFile.Close()

	go io.Copy(s.ptyFile, channel)
	go io.Copy(channel, s.ptyFile)

	shell := s.ptyFile.Command("powershell.exe")
	if err := shell.Start(); err != nil {
		lg := logger.GetLogger()
		lg.Error("Failed to start shell: %v", err)
		return
	}

	if err := shell.Wait(); err != nil {
		lg := logger.GetLogger()
		lg.Error("Shell exited with error: %v", err)
	}

	shell.Process.Kill()
}

func NewShell() (sshShell, error) {
	ptyFile, err := pty.New()
	if err != nil {
		return nil, err
	}

	return &winShell{
		ptyFile: ptyFile,
		columns: 80,
		rows:    24,
	}, nil
}
