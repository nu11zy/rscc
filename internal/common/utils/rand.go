package utils

import "math/rand"

// RandInt returns random value in range [0, max)
func RandInt(max int) int {
	return rand.Intn(max)
}
