package main

// utils.go — shared helper functions for parsing and formatting.

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// parseChainID parses a chain ID string (decimal or "0x" hex).
func parseChainID(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 1
	}
	if strings.HasPrefix(s, "0x") {
		n, err := strconv.ParseInt(s[2:], 16, 64)
		if err != nil {
			return 1
		}
		return n
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 1
	}
	return n
}

// hexDecode decodes a hex string (with or without 0x prefix).
func hexDecode(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return nil, nil
	}
	if len(s)%2 != 0 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}

// etherToWei converts a float amount in ether to wei (big.Int).
func etherToWei(ether *big.Float) *big.Int {
	weiPerEther := new(big.Float).SetFloat64(1e18)
	weiFloat := new(big.Float).Mul(ether, weiPerEther)
	wei, _ := weiFloat.Int(nil)
	return wei
}

// weiToGweiFloat converts wei to gwei as a float.
func weiToGweiFloat(wei *big.Int) float64 {
	if wei == nil {
		return 0
	}
	gwei := new(big.Float).Quo(
		new(big.Float).SetInt(wei),
		new(big.Float).SetFloat64(1e9),
	)
	f, _ := gwei.Float64()
	return f
}

// formatHex ensures a 0x prefix.
func formatHex(s string) string {
	if !strings.HasPrefix(s, "0x") {
		return "0x" + s
	}
	return s
}

// bigIntToString safely converts a big.Int to string.
func bigIntToString(n *big.Int) string {
	if n == nil {
		return "0"
	}
	return n.String()
}

// errString returns the error message or empty string.
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// ensure fmt is used
var _ = fmt.Sprintf
