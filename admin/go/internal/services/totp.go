// Real RFC 6238 TOTP (HMAC-SHA1, 30-second step, 6 digits).
//
// The previous Verify2FACode accepted any 6-digit string, which made 2FA
// unenforceable. This file provides genuine TOTP verification.

package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"time"
)

// GenerateTOTPSecret returns a base32-encoded 20-byte (160-bit) secret, the
// size recommended by RFC 4226 for HMAC-SHA1.
func GenerateTOTPSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}

// verifyTOTPCode validates a 6-digit code against the secret, allowing one step
// of clock drift in each direction.
func verifyTOTPCode(secret, code string) bool {
	if len(code) != 6 {
		return false
	}
	key, err := decodeTOTPSecret(secret)
	if err != nil {
		return false
	}
	now := time.Now().Unix()
	for _, step := range []int64{0, -1, 1} {
		if totpCode(key, now/30+step) == code {
			return true
		}
	}
	return false
}

// decodeTOTPSecret accepts both padded and unpadded base32 secrets.
func decodeTOTPSecret(secret string) ([]byte, error) {
	if key, err := base32.StdEncoding.DecodeString(secret); err == nil {
		return key, nil
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
}

func totpCode(key []byte, counter int64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(counter))
	mac := hmac.New(sha1.New, key)
	mac.Write(buf[:])
	sum := mac.Sum(nil)

	offset := int(sum[len(sum)-1] & 0x0f)
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	digits := value % 1000000

	out := []byte("000000")
	for i := 5; i >= 0; i-- {
		out[i] = byte('0' + (digits % 10))
		digits /= 10
	}
	return string(out)
}
