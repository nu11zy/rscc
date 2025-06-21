//go:build execass && windows
// +build execass,windows

package subsystems

import "agent/internal/sshd/subsystems/execass"

func init() {
	Subsystems["execass"] = execass.Start
}
