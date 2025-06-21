//go:build pscan
// +build pscan

package subsystems

import "agent/internal/sshd/subsystems/pscan"

func init() {
	Subsystems["pscan"] = pscan.Start
}
