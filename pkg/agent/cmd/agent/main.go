package main

import (
	"agent"
	"agent/internal/logger"
	"agent/internal/metadata"
	"agent/internal/sshd"
)

func main() {
	lg := logger.GetLogger()
	if len(agent.EKey) == 0 {
		lg.Fatal("Encryption key is not set")
	}
	if len(agent.PrivateKey) == 0 {
		lg.Fatal("Private key is not set")
	}
	if len(agent.Fingerprint) == 0 {
		lg.Fatal("Fingerprint is not set")
	}
	if len(agent.Servers) == 0 {
		lg.Fatal("Servers are not set")
	}
	if len(agent.SSHClient) == 0 {
		lg.Fatal("SSH client is not set")
	}

	metadata := metadata.GetMetadata()
	lg.Info("Metadata: %s", metadata)

	agentConfig := sshd.NewAgentConfig(metadata)
	if err := agentConfig.Start(); err != nil {
		lg.Error("Agent error: %v", err)
	}
}
