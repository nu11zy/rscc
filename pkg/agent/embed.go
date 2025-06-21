package agent

import (
	_ "embed"
)

//go:embed _private_key
var PrivateKey []byte

//go:embed _fingerprint
var Fingerprint []byte

//go:embed _servers
var Servers []byte

//go:embed _encryption_key
var EKey []byte

//go:embed _ssh_client
var SSHClient []byte
