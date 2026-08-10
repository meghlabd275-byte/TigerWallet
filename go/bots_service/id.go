package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
)

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func jsonDecode(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func splitPair(pair string) []string {
	p := strings.Split(strings.ToLower(strings.TrimSpace(pair)), "/")
	if len(p) != 2 || p[0] == "" || p[1] == "" {
		return nil
	}
	return p
}
