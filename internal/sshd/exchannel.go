package sshd

import "golang.org/x/crypto/ssh"

type ExitStatus struct {
	Status uint32
}

type ExtendedChannel struct {
	ssh.Channel

	closed bool
}

func NewExtendedChannel(channel ssh.Channel) *ExtendedChannel {
	return &ExtendedChannel{
		Channel: channel,
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
