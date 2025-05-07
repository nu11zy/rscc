package sshd

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

type ExitStatus struct {
	Status uint32
}

type ExtendedChannel struct {
	ssh.Channel

	closed bool
}

func NewExtendedChannel(channel ssh.Channel) *ExtendedChannel {
	return &ExtendedChannel{Channel: channel}
}

func (e *ExtendedChannel) CloseWithStatus(status int) {
	if e.closed {
		return
	}
	e.SendRequest("exit-status", false, ssh.Marshal(&ExitStatus{Status: uint32(status)}))
	e.Close()
	e.closed = true
}

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
