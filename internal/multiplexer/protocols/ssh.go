package protocols

import "bytes"

const Ssh Type = "ssh"

type SshProto struct {
	_ Proto
}

func NewSshProto() SshProto {
	return SshProto{}
}

func (s SshProto) IsProto(data []byte) bool {
	return bytes.HasPrefix(data, []byte{'S', 'S', 'H'})
}

func (s SshProto) Type() Type {
	return Ssh
}

func (s SshProto) IsUnwrapped() bool {
	return true
}

func init() {
	protocols[Ssh] = SshProto{}
}
