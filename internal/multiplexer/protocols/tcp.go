package protocols

import (
	"bytes"
)

const TcpDownload Type = "tcp-download"

type TcpDownloadProto struct {
	_ Proto
}

func NewTcpDownloadProto() TcpDownloadProto {
	return TcpDownloadProto{}
}

func (s TcpDownloadProto) IsProto(data []byte) bool {
	return bytes.HasPrefix(data, []byte{'R', 'S', 'C', 'C'})
}

func (s TcpDownloadProto) Type() Type {
	return TcpDownload
}

func (s TcpDownloadProto) IsUnwrapped() bool {
	return true
}

func init() {
	protocols[TcpDownload] = TcpDownloadProto{}
}
