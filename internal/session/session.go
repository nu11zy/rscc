package session

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Agent metadata
type Metadata struct {
	Username string   `json:"u,omitempty"`
	Hostname string   `json:"h,omitempty"`
	Domain   string   `json:"d,omitempty"`
	IPs      []string `json:"i,omitempty"`
	OSMeta   string   `json:"om,omitempty"`
	ProcName string   `json:"pn,omitempty"`
	IsPriv   bool     `json:"ip,omitempty"`
	Extra    string   `json:"e,omitempty"`
}

type Session struct {
	ID         string
	CreatedAt  time.Time
	Metadata   Metadata
	RemoteAddr string
	SSHConn    *ssh.ServerConn
}

func NewSession(encMetadata string, sshConn *ssh.ServerConn) (*Session, error) {
	jsonMetadata, err := base64.RawStdEncoding.DecodeString(encMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	var metadata Metadata
	if err = json.Unmarshal(jsonMetadata, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &Session{
		Metadata:   metadata,
		RemoteAddr: strings.Split(sshConn.RemoteAddr().String(), ":")[0],
		SSHConn:    sshConn,
	}, nil
}
