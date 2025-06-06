package protocols

import "net/http"

var httpProtocol = Protocol{
	Name: "http",
	Header: [][]byte{
		[]byte(http.MethodConnect),
		[]byte(http.MethodDelete),
		[]byte(http.MethodGet),
		[]byte(http.MethodHead),
		[]byte(http.MethodOptions),
		[]byte(http.MethodPatch),
		[]byte(http.MethodPost),
		[]byte(http.MethodPut),
		[]byte(http.MethodTrace),
	},
	IsUnwrapped: true,
}

func init() {
	protocols = append(protocols, httpProtocol)
}
