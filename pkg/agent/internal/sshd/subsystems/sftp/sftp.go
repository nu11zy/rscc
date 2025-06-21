//go:build sftp
// +build sftp

package sftp

import (
	"agent/internal/logger"

	realsftp "github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func Start(channel ssh.Channel, args []string) {
	defer channel.Close()

	lg := logger.GetLogger()
	lg.Info("SFTP subsystem request received")

	server, err := realsftp.NewServer(channel)
	if err != nil {
		lg.Error("Failed to create SFTP server: %v", err)
		return
	}

	err = server.Serve()
	if err != nil {
		lg.Error("SFTP server error: %v", err)
	}
}
