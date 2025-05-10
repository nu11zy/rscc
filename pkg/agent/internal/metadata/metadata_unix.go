//go:build !windows
// +build !windows

package metadata

import (
	"strings"

	"golang.org/x/sys/unix"
)

func getOSMeta() string {
	u := unix.Utsname{}
	if err := unix.Uname(&u); err != nil {
		return "<unknown>"
	}

	return strings.TrimRight(string(u.Version[:]), "\x00")
}

func isPrivileged() bool {
	return unix.Getuid() == 0
}
