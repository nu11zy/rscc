package subsystems

import (
	"agent/internal/logger"
	"fmt"

	"golang.org/x/crypto/ssh"
)

func init() {
	Subsystems["list"] = listSubsystems
}

func listSubsystems(channel ssh.Channel, args []string) {
	lg := logger.GetLogger()
	defer channel.Close()

	lg.Info("List subsystems request received")

	for k := range Subsystems {
		channel.Write([]byte(fmt.Sprintf("- %s\n", k)))
	}
}
