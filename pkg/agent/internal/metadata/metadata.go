package metadata

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"log"
	"os"
	"os/user"
)

type Metadata struct {
	Username string
	Hostname string
}

func GetMetadata() (string, error) {
	metadata := &Metadata{
		Username: getUsername(),
		Hostname: getHostname(),
	}

	encoded, err := EncodeMetadata(metadata)
	if err != nil {
		return "", err
	}

	log.Printf("Metadata: %v", encoded)
	return encoded, nil
}

func EncodeMetadata(metadata *Metadata) (string, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(metadata)
	if err != nil {
		return "", err
	}
	encoded := base64.URLEncoding.EncodeToString(buf.Bytes())
	return encoded, nil
}

func getUsername() string {
	u, err := user.Current()
	if err != nil {
		return "<unknown>"
	}
	return u.Username
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "<unknown>"
	}
	return hostname
}
