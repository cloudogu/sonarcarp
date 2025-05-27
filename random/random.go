package random

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func String(length int) (string, error) {
	charset := "abcedfghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random int: %w", err)
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}
