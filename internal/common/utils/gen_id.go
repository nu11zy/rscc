package utils

import (
	"crypto/rand"
	"math/big"
	"rscc/internal/common/constants"
)

const safeCharset = "abcdefghjkmnpqrstuvwxyz1234567890"

func GenID() string {
	b := make([]byte, constants.IDLength)
	for i := 0; i < constants.IDLength; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(safeCharset))))
		if err != nil {
			panic("failed to generate secure random ID")
		}
		b[i] = safeCharset[n.Int64()]
	}
	return string(b)
}
