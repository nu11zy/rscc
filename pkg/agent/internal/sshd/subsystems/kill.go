//go:build kill
// +build kill

package subsystems

import "agent/internal/sshd/subsystems/kill"

func init() {
	Subsystems["kill"] = kill.Start
}
