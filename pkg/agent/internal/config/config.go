package config

import (
	"agent"
	"agent/internal/crypt"
	"agent/internal/logger"
	"os"
	"strings"
)

func GetServers() []string {
	lg := logger.GetLogger()

	var servers []string
	if os.Getenv("SRV") != "" {
		servers = strings.Split(os.Getenv("SRV"), ",")
		return servers
	}

	decodedBytes, err := crypt.Decrypt(agent.EKey, agent.Servers)
	if err != nil {
		lg.Fatal("Failed to decrypt servers: %v", err)
	}

	servers = strings.Split(string(decodedBytes), ",")
	return servers
}

func GetPrivateKey() []byte {
	lg := logger.GetLogger()

	decodedBytes, err := crypt.Decrypt(agent.EKey, agent.PrivateKey)
	if err != nil {
		lg.Fatal("Failed to decrypt private key: %v", err)
	}

	return decodedBytes
}

func GetFingerprint() []byte {
	lg := logger.GetLogger()

	decodedBytes, err := crypt.Decrypt(agent.EKey, agent.Fingerprint)
	if err != nil {
		lg.Fatal("Failed to decrypt fingerprint: %v", err)
	}

	return decodedBytes
}

func GetSSHClient() string {
	lg := logger.GetLogger()

	decodedBytes, err := crypt.Decrypt(agent.EKey, agent.SSHClient)
	if err != nil {
		lg.Fatal("Failed to decrypt SSH client: %v", err)
	}

	return string(decodedBytes)
}
