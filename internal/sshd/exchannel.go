package sshd

import "golang.org/x/crypto/ssh"

type ExitStatus struct {
	Status uint32
}

type OperatorSession struct {
	Username    string
	Permissions *ssh.Permissions
}

type ExtendedChannel struct {
	ssh.Channel

	Operator *OperatorSession
	closed   bool
}

func NewExtendedChannel(channel ssh.Channel, operator *OperatorSession) *ExtendedChannel {
	return &ExtendedChannel{
		Channel:  channel,
		Operator: operator,
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
