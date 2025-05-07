package opsrv

import "golang.org/x/crypto/ssh"

type ExitStatus struct {
	Status uint32
}

type OperatorSession struct {
	username    string
	permissions *ssh.Permissions
}

type ExtendedChannel struct {
	ssh.Channel

	operator *OperatorSession
	closed   bool
}

func NewExtendedChannel(channel ssh.Channel, operator *OperatorSession) *ExtendedChannel {
	return &ExtendedChannel{
		Channel:  channel,
		operator: operator,
	}
}

func (e *ExtendedChannel) CloseWithStatus(status int) {
	if e.closed {
		return
	}
	e.SendRequest("exit-status", false, ssh.Marshal(&ExitStatus{Status: uint32(status)}))
	e.Close()
	e.closed = true
}
