package myproc

import "strings"

var Hash = fnv1a

const (
	FNV1_OFFSET_BASIS = 0x811c9dc5
	FNV1_PRIME        = 0x01000193
)

// fnv1a hashing algorithm
func fnv1a(str string) uint32 {
	str = strings.ToLower(str)
	var hash uint32 = FNV1_OFFSET_BASIS
	for i := 0; i < len(str); i++ {
		hash ^= uint32(str[i])
		hash *= FNV1_PRIME
	}
	return hash
}
