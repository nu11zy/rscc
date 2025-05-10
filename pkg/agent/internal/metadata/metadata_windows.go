package metadata

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func getOSMeta() string {
	v := *windows.RtlGetVersion()
	return fmt.Sprintf("Windows %d.%d.%d", v.MajorVersion, v.MinorVersion, v.BuildNumber)
}

func isPrivileged() bool {
	var sid *windows.SID
	if err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid); err != nil {
		return false
	}

	if _, err := windows.Token(0).IsMember(sid); err != nil {
		return false
	}

	return true
}
