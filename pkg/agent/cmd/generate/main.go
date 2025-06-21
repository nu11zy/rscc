//go:build exclude
// +build exclude

package main

import (
	"agent/internal/crypt"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	mrand "math/rand"
	"os"
)

func main() {
	servers := os.Getenv("SERVERS")
	privateKey := os.Getenv("PRIV_KEY")
	fingerprint := os.Getenv("FINGERPRINT")
	sshClient := os.Getenv("SSH_CLIENT")

	if servers == "" || privateKey == "" || fingerprint == "" || sshClient == "" {
		fmt.Println("SERVERS, PRIV_KEY, FINGERPRINT, and SSH_CLIENT must be set")
		os.Exit(1)
	}

	// Generate encryption key
	encryptionKey := make([]byte, 16+mrand.Intn(256))
	rand.Read(encryptionKey)

	// Encrypt servers
	encryptedServers, err := crypt.Encrypt(encryptionKey, []byte(servers))
	if err != nil {
		fmt.Println("Error encrypting servers: ", err)
		os.Exit(1)
	}

	// Encrypt private key
	decodedPrivateKey, err := base64.RawStdEncoding.DecodeString(privateKey)
	if err != nil {
		fmt.Println("Error decoding private key: ", err)
		os.Exit(1)
	}
	encryptedPrivateKey, err := crypt.Encrypt(encryptionKey, decodedPrivateKey)
	if err != nil {
		fmt.Println("Error encrypting private key: ", err)
		os.Exit(1)
	}

	// Encrypt fingerprint
	encryptedFingerprint, err := crypt.Encrypt(encryptionKey, []byte(fingerprint))
	if err != nil {
		fmt.Println("Error encrypting fingerprint: ", err)
		os.Exit(1)
	}

	// Encrypt SSH client
	encryptedSshClient, err := crypt.Encrypt(encryptionKey, []byte(sshClient))
	if err != nil {
		fmt.Println("Error encrypting SSH client: ", err)
		os.Exit(1)
	}

	// Save encrypted data to file
	err = os.WriteFile("_servers", encryptedServers, 0644)
	if err != nil {
		fmt.Println("Error saving encrypted servers: ", err)
		os.Exit(1)
	}
	err = os.WriteFile("_private_key", encryptedPrivateKey, 0644)
	if err != nil {
		fmt.Println("Error saving encrypted private key: ", err)
		os.Exit(1)
	}
	err = os.WriteFile("_encryption_key", encryptionKey, 0644)
	if err != nil {
		fmt.Println("Error saving encryption key: ", err)
		os.Exit(1)
	}
	err = os.WriteFile("_fingerprint", encryptedFingerprint, 0644)
	if err != nil {
		fmt.Println("Error saving encrypted fingerprint: ", err)
		os.Exit(1)
	}
	err = os.WriteFile("_ssh_client", encryptedSshClient, 0644)
	if err != nil {
		fmt.Println("Error saving encrypted SSH client: ", err)
		os.Exit(1)
	}

	// Print success message
	fmt.Println("Successfully encrypted data")
}
