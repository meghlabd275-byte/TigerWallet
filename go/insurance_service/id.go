package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
)

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func fmtSscan(s string, v *float64) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		f = 0
	}
	*v = f
}

func premiumStr(f float64) string {
	return fmt.Sprintf("%.0f", f)
}
