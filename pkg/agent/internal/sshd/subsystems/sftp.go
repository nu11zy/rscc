//go:build sftp
// +build sftp

package subsystems

import (
	"log"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func init() {
	Subsystems["sftp"] = subsystemSFTP
}

func subsystemSFTP(channel ssh.Channel, args []string) {
	defer channel.Close()

	// {{if .Debug}}
	log.Printf("SFTP subsystem request received")
	// {{end}}

	server, err := sftp.NewServer(channel)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to create SFTP server: %v", err)
		// {{end}}
		return
	}

	err = server.Serve()
	if err != nil {
		// {{if .Debug}}
		log.Printf("SFTP server error: %v", err)
		// {{end}}
	}
}
