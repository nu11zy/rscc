package protocols

var sshProtocol = Protocol{
	Name:        "ssh",
	Header:      [][]byte{{'S', 'S', 'H'}},
	IsUnwrapped: true,
}

func init() {
	protocols = append(protocols, sshProtocol)
}
