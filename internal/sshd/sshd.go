package sshd

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

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
