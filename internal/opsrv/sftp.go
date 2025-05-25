package opsrv

import (
	"os"
	"path/filepath"
	"rscc/internal/sshd"

	"go.uber.org/zap"

	"github.com/pkg/sftp"
)

func sftpHandler(lg *zap.SugaredLogger, channel *sshd.ExtendedChannel) {
	defer channel.CloseWithStatus(0)

	currentDir, err := os.Getwd()
	if err != nil {
		lg.Errorf("Failed to get current directory: %v", err)
		return
	}

	server, err := sftp.NewServer(
		channel,
		sftp.WithServerWorkingDirectory(filepath.Join(currentDir, "agents")),
	)
	if err != nil {
		lg.Errorf("Failed to create SFTP server: %v", err)
		return
	}

	if err := server.Serve(); err != nil {
		lg.Errorf("Failed to serve SFTP server: %v", err)
		return
	}
}
