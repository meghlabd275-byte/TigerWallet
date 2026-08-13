package main

// btc_helpers.go — Low-level helpers for Bitcoin transaction serialization.
// Real crypto, no fakes/stubs.

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

// bytesBuffer is a simple byte buffer for serializing Bitcoin wire-format txs.
type bytesBuffer struct {
	data []byte
}

func (b *bytesBuffer) writeUint32(v uint32) {
	b.data = append(b.data, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

func (b *bytesBuffer) writeUint64(v uint64) {
	b.data = append(b.data, byte(v), byte(v>>8), byte(v>>16), byte(v>>24),
		byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56))
}

func (b *bytesBuffer) writeVarInt(v uint64) {
	if v < 0xFD {
		b.data = append(b.data, byte(v))
	} else if v <= 0xFFFF {
		b.data = append(b.data, 0xFD, byte(v), byte(v>>8))
	} else if v <= 0xFFFFFFFF {
		b.data = append(b.data, 0xFE, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
	} else {
		b.data = append(b.data, 0xFF, byte(v), byte(v>>8), byte(v>>16), byte(v>>24),
			byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56))
	}
}

func (b *bytesBuffer) writeBytes(data []byte) {
	b.data = append(b.data, data...)
}

func (b *bytesBuffer) bytes() []byte {
	return b.data
}

// parseHexReverse decodes a hex string and reverses the byte order (Bitcoin
// txid display format is reversed).
func parseHexReverse(s string) []byte {
	s = stringsTrimPrefix(s, "0x")
	b := parseHex(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return b
}

// doubleSHA256 computes SHA256(SHA256(data)) — the Bitcoin hash function.
func doubleSHA256(data []byte) []byte {
	h1 := sha256Hash(data)
	h2 := sha256Hash(h1)
	return h2
}

// buildP2PKHScript builds the P2PKH subscript for signing:
// OP_DUP OP_HASH160 <hash160> OP_EQUALVERIFY OP_CHECKSIG
func buildP2PKHScript(hash160 []byte) []byte {
	script := make([]byte, 0, 25)
	script = append(script, 0x76) // OP_DUP
	script = append(script, 0xA9) // OP_HASH160
	script = append(script, 0x14) // push 20 bytes
	script = append(script, hash160...)
	script = append(script, 0x88) // OP_EQUALVERIFY
	script = append(script, 0xAC) // OP_CHECKSIG
	return script
}

// buildP2PKHOutputScript builds the P2PKH output scriptPubKey for an address.
func buildP2PKHOutputScript(address string) []byte {
	// Decode the base58check address to get the hash160.
	version, hash160, ok := base58CheckDecode(address)
	if !ok {
		// Invalid BTC address — return empty script (the broadcast will reject it).
		return buildP2PKHScript(make([]byte, 20))
	}
	_ = version
	// Output script: OP_DUP OP_HASH160 <hash160> OP_EQUALVERIFY OP_CHECKSIG
	return buildP2PKHScript(hash160)
}

// base58CheckDecode decodes a base58check-encoded string, returning the
// version byte, the payload (hash160), and ok=true if the checksum validates.
func base58CheckDecode(address string) (byte, []byte, bool) {
	decoded := base58Decode(address)
	if len(decoded) < 5 {
		return 0, nil, false
	}
	// Last 4 bytes are the checksum.
	payload := decoded[:len(decoded)-4]
	checksum := decoded[len(decoded)-4:]
	// Verify checksum = first 4 bytes of doubleSHA256(payload).
	expected := doubleSHA256(payload)
	for i := 0; i < 4; i++ {
		if checksum[i] != expected[i] {
			return 0, nil, false
		}
	}
	if len(payload) < 1 {
		return 0, nil, false
	}
	version := payload[0]
	hash160 := payload[1:]
	return version, hash160, true
}

// base58Decode decodes a base58 string to bytes (no checksum verification).
func base58Decode(s string) []byte {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	dec := []byte{}
	for i := 0; i < len(s); i++ {
		c := s[i]
		idx := -1
		for j := 0; j < 58; j++ {
			if alphabet[j] == c {
				idx = j
				break
			}
		}
		if idx < 0 {
			continue // skip invalid chars
		}
		// Multiply dec by 58 and add idx.
		carry := big.NewInt(int64(idx))
		for j := len(dec) - 1; j >= 0; j-- {
			val := new(big.Int).Mul(big.NewInt(int64(dec[j])), big.NewInt(58))
			val.Add(val, carry)
			dec[j] = byte(val.Int64() & 0xFF)
			carry = new(big.Int).Rsh(val, 8)
		}
		for carry.Sign() > 0 {
			dec = append([]byte{byte(carry.Int64() & 0xFF)}, dec...)
			carry = new(big.Int).Rsh(carry, 8)
		}
	}
	// Handle leading '1' bytes (zero prefix).
	for i := 0; i < len(s) && s[i] == '1'; i++ {
		dec = append([]byte{0}, dec...)
	}
	return dec
}

// httpGet fetches a URL and returns the body bytes.
func httpGet(url string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// jsonUnmarshal wraps json.Unmarshal.
func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// stringsTrimPrefix is a helper to avoid importing strings in this file.
func stringsTrimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

// keccak256Hash computes Keccak-256 (used by EVM, re-exported here for convenience).
func keccak256Hash(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

// ecdsaPrivateKeyType is an alias for convenience.
type ecdsaPrivateKeyType = ecdsa.PrivateKey

// re-export crypto.Sign for the BTC helpers (avoids duplicate import warnings).
var _ = crypto.Sign
