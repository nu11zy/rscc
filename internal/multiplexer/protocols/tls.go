package protocols

import (
	"bytes"
)

const Tls Type = "tls"

type TlsProto struct {
	_ Proto
}

func NewTlsProto() TlsProto {
	return TlsProto{}
}

func (s TlsProto) IsProto(data []byte) bool {
	// 0x16 - handshake record in TLS handshake
	// 0x03 0x01 - protocol version 3.1 (TLS 1.0). Same in TLS 1.0/1.2/1.3
	return bytes.HasPrefix(data, []byte{0x16, 0x03, 0x01})
}

func (s TlsProto) Type() Type {
	return Tls
}

func (s TlsProto) IsUnwrapped() bool {
	return false
}

func init() {
	protocols[Tls] = TlsProto{}
}
