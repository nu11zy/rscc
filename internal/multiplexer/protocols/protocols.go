package protocols

/*
	2 types of protocols:
	- Wrapped (tls)
	- Unwrapped (ssh/tcp-download)
*/

type Type string
type Proto interface {
	// IsProto returns true if first N bytes corresponds to protocol
	IsProto([]byte) bool
	// Type returns type of protocol
	Type() Type
	// IsUnwrapped returns true if protocol if fully unwrapped
	IsUnwrapped() bool
}

var protocols = make(map[Type]Proto)

// IsUnwraped returns true if supplied protocol type is fully unwrapped
func IsUnwrapped(proto Type) bool {
	return false
}
