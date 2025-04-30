package utils

import (
	"crypto/rand"
	"math/big"
)

const safeCharset = "abcdefghjkmnpqrstuvwxyz1234567890"
const idLength = 8

func GenID() string {
	b := make([]byte, idLength)
	for i := 0; i < idLength; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(safeCharset))))
		if err != nil {
			panic("failed to generate secure random ID")
		}
		b[i] = safeCharset[n.Int64()]
	}
	return string(b)
}
