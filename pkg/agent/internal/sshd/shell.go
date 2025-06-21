package sshd

import (
	"encoding/binary"
	"errors"

	"golang.org/x/crypto/ssh"
)

type sshShell interface {
	SetSize(req *ssh.Request) error
	HandleShell(channel ssh.Channel)
}

type ptyData struct {
	Term    string
	Columns uint32
	Rows    uint32
	Width   uint32
	Height  uint32
	Modes   string
}

func ParseSize(req *ssh.Request) (uint32, uint32, error) {
	switch req.Type {
	case "pty-req":
		var data ptyData
		if err := ssh.Unmarshal(req.Payload, &data); err != nil {
			return 0, 0, err
		}
		return data.Columns, data.Rows, nil
	case "window-change":
		columns := binary.BigEndian.Uint32(req.Payload)
		rows := binary.BigEndian.Uint32(req.Payload[4:])
		return columns, rows, nil
	default:
		return 0, 0, errors.New("unknown resize request type")
	}
}
