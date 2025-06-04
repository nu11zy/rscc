package opsrv

import (
	"path/filepath"
	"rscc/internal/sshd"

	"go.uber.org/zap"

	"github.com/pkg/sftp"
)

func sftpHandler(lg *zap.SugaredLogger, channel *sshd.ExtendedChannel, dataPath string) {
	defer channel.CloseWithStatus(0)

	server, err := sftp.NewServer(
		channel,
		sftp.WithServerWorkingDirectory(filepath.Join(dataPath, "agents")),
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
