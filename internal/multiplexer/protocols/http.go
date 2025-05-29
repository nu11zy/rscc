package protocols

import (
	"bytes"
	"net/http"
)

const HttpDownload Type = "http-download"

type HttpDownloadProto struct {
	_ Proto
}

func NewHttpDownloadProto() HttpDownloadProto {
	return HttpDownloadProto{}
}

func (s HttpDownloadProto) IsProto(data []byte) bool {
	// list of base HTTP methods
	methods := [][]byte{
		[]byte(http.MethodConnect),
		[]byte(http.MethodDelete),
		[]byte(http.MethodGet),
		[]byte(http.MethodHead),
		[]byte(http.MethodOptions),
		[]byte(http.MethodPatch),
		[]byte(http.MethodPost),
		[]byte(http.MethodPut),
		[]byte(http.MethodTrace),
	}
	// check if first bytes are valid HTTP methods
	for _, method := range methods {
		if bytes.HasPrefix(data, method) {
			return true
		}
	}
	return false
}

func (s HttpDownloadProto) Type() Type {
	return HttpDownload
}

func (s HttpDownloadProto) IsUnwrapped() bool {
	return true
}

func init() {
	protocols[HttpDownload] = HttpDownloadProto{}
}
