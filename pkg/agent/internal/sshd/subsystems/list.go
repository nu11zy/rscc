package subsystems

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

func init() {
	Subsystems["list"] = subsystemList
}

// sybsystemList prints supported subsystems by agent
func subsystemList(channel ssh.Channel, _ []string) {
	defer channel.Close()

	for k := range Subsystems {
		channel.Write([]byte(fmt.Sprintf("- %s\n", k)))
	}
}
