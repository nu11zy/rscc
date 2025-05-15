package subsystems

import (
	"golang.org/x/crypto/ssh"
)

var Subsystems = make(map[string]func(ssh.Channel, []string))
