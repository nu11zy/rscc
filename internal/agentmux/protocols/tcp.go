package protocols

var tcpProtocol = Protocol{
	Name:        "tcp",
	Header:      [][]byte{{'R', 'S', 'C', 'C'}},
	IsUnwrapped: true,
}

func init() {
	protocols = append(protocols, tcpProtocol)
}
