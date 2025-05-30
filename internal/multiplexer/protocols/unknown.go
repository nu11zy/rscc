package protocols

const Unknown Type = "unknown"

type UnknownProto struct {
	_ Proto
}

func NewUnknownProto() UnknownProto {
	return UnknownProto{}
}

func (s UnknownProto) IsProto(data []byte) bool {
	return false
}

func (s UnknownProto) Type() Type {
	return Unknown
}

func (s UnknownProto) IsUnwrapped() bool {
	return false
}

func init() {
	protocols[Unknown] = UnknownProto{}
}
