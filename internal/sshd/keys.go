package sshd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

type ECDSAKey struct {
	curve      elliptic.Curve
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
}

func NewECDSAKey() (*ECDSAKey, error) {
	curve := elliptic.P384()
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}

	return &ECDSAKey{
		curve:      curve,
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}, nil
}

// GetPublicKey returns the public key in SSH format
func (k *ECDSAKey) GetPublicKey() ([]byte, error) {
	sshPubKey, err := ssh.NewPublicKey(k.publicKey)
	if err != nil {
		return nil, fmt.Errorf("new public key: %w", err)
	}

	return ssh.MarshalAuthorizedKey(sshPubKey), nil
}

// GetFingerprint returns the fingerprint of the public key
func (k *ECDSAKey) GetFingerprint() (string, error) {
	sshPubKey, err := ssh.NewPublicKey(k.publicKey)
	if err != nil {
		return "", fmt.Errorf("new public key: %w", err)
	}
	return ssh.FingerprintSHA256(sshPubKey), nil
}

// GetPrivateKey returns the private key in PEM format
func (k *ECDSAKey) GetPrivateKey() ([]byte, error) {
	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(k.privateKey)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}

	pemKey := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyBytes,
	}

	return pem.EncodeToMemory(pemKey), nil
}
