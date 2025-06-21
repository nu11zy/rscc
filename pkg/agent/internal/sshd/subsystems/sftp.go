//go:build sftp
// +build sftp

package subsystems

import "agent/internal/sshd/subsystems/sftp"

func init() {
	Subsystems["sftp"] = sftp.Start
}
