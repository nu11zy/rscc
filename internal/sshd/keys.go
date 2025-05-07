package sshd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

func GenerateRSAPrivateKey() ([]byte, error) {
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

func GenerateRSAPublicKey(privateKey []byte) ([]byte, error) {
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, fmt.Errorf("failed to decode private key")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return x509.MarshalPKIXPublicKey(key.Public())
}

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
