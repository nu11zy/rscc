//go:build kill
// +build kill

package kill

import (
	"agent/internal/logger"
	"os"

	"golang.org/x/crypto/ssh"
)

func Start(channel ssh.Channel, args []string) {
	defer channel.Close()

	lg := logger.GetLogger()
	lg.Info("Kill subsystem request received")

	channel.Close()
	os.Exit(0)
}
