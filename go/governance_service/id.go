package main

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
)

// newID returns a 16-byte hex random identifier.
func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// cmpBigInt compares two decimal-string big integers. Returns -1, 0, or 1.
// Malformed inputs are treated as zero.
func cmpBigInt(a, b string) int {
	ai, ok := new(big.Int).SetString(a, 10)
	if !ok {
		ai = new(big.Int)
	}
	bi, ok := new(big.Int).SetString(b, 10)
	if !ok {
		bi = new(big.Int)
	}
	return ai.Cmp(bi)
}
