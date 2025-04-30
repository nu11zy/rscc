package sshd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

func GeneratePrivateKey() ([]byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	pemKey := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	pemBytes := pem.EncodeToMemory(pemKey)
	return pemBytes, nil
}
