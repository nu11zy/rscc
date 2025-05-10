//go:build kill
// +build kill

package subsystems

import (
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

func init() {
	Subsystems["kill"] = subsystemKill
}

func subsystemKill(channel ssh.Channel, args []string) {
	defer channel.Close()

	// {{if .Debug}}
	log.Printf("Kill subsystem request received")
	// {{end}}

	channel.Close()
	os.Exit(0)
}
