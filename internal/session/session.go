package session

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"

	"rscc/internal/common/utils"

	"golang.org/x/crypto/ssh"
)

// Agent metadata
type Metadata struct {
	Username string
	Hostname string
}

type Session struct {
	ID       string
	Metadata Metadata
	SSHConn  *ssh.ServerConn
}

func NewSession(encMetadata string, sshConn *ssh.ServerConn) (*Session, error) {
	rawMetadata, err := base64.URLEncoding.DecodeString(encMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	var metadata Metadata
	dec := gob.NewDecoder(bytes.NewReader(rawMetadata))
	err = dec.Decode(&metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize metadata: %w", err)
	}

	id := utils.GenID()
	return &Session{
		ID:       id,
		Metadata: metadata,
		SSHConn:  sshConn,
	}, nil
}
