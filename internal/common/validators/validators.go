package validators

import (
	"net"
	"regexp"
	"slices"
	"strconv"
)

func ValidateAddr(addr string) bool {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}

	return ValidateHost(host) && ValidatePort(port)
}

func ValidateHost(host string) bool {
	// Check if ip address
	ip := net.ParseIP(host)
	if ip != nil {
		return true
	}

	// Check if hostname
	hostnameRegex := `^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)+([A-Za-z]|[A-Za-z][A-Za-z0-9\-]*[A-Za-z0-9])$`
	_, err := regexp.MatchString(hostnameRegex, host)
	if err != nil {
		return false
	}

	return true
}

func ValidatePort(port any) bool {
	switch port := port.(type) {
	case string:
		portInt, err := strconv.Atoi(port)
		if err != nil {
			return false
		}
		if portInt < 1 || portInt > 65535 {
			return false
		}
		return true
	case int:
		if port < 1 || port > 65535 {
			return false
		}
		return true
	default:
		return false
	}
}
func ValidateGOOS(goos string) bool {
	var validGOOS = []string{"windows", "linux", "darwin"}
	return slices.Contains(validGOOS, goos)
}

func ValidateGOARCH(goarch string) bool {
	var validGOARCH = []string{"amd64", "arm64"}
	return slices.Contains(validGOARCH, goarch)
}
