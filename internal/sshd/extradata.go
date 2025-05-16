package sshd

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/ssh"
)

type ExtraData struct {
	TargetHost     string
	TargetPort     uint32
	OriginatorIP   string
	OriginatorPort uint32
}

func GetExtraData(extraData []byte) (*ExtraData, error) {
	var data ExtraData
	if err := ssh.Unmarshal(extraData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal extra data: %w", err)
	}
	return &data, nil
}

type PtyReq struct {
	Term          string
	Columns, Rows uint32
	Width, Height uint32
	Modes         string
}

func ParsePtyReq(req *ssh.Request) (*PtyReq, error) {
	var data PtyReq
	if err := ssh.Unmarshal(req.Payload, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pty request: %w", err)
	}
	return &data, nil
}

func ParseWindowChangeReq(req []byte) (uint32, uint32) {
	columns := binary.BigEndian.Uint32(req)
	rows := binary.BigEndian.Uint32(req[4:])
	return columns, rows
}
