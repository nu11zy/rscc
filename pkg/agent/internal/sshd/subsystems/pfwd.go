//go:build pfwd
// +build pfwd

package subsystems

import "agent/internal/sshd/subsystems/pfwd"

func init() {
	Subsystems["pfwd"] = pfwd.Start
}
