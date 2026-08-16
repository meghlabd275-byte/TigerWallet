package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"time"
)

// Real RFC 6238 TOTP (Time-based One-Time Password), HMAC-SHA1, 30s step,
// 6 digits. Used for admin 2FA. No length-check stubs.
//
// generateTOTPSecret returns a base32-encoded 20-byte secret.
func generateTOTPSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(b), nil
}

// verifyTOTP validates a 6-digit code against the secret within ±1 step window.
func verifyTOTP(secret, code string) bool {
	if len(code) != 6 {
		return false
	}
	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return false
	}
	now := time.Now().Unix()
	for _, step := range []int64{0, -1, 1} {
		if totp(key, now/30+step) == code {
			return true
		}
	}
	return false
}

func totp(key []byte, counter int64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(counter))
	mac := hmac.New(sha1.New, key)
	mac.Write(buf[:])
	sum := mac.Sum(nil)
	offset := int(sum[len(sum)-1] & 0x0f)
	code := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	mod := uint32(1000000)
	digits := code % mod
	out := []byte("000000")
	for i := 5; i >= 0; i-- {
		out[i] = byte('0' + (digits % 10))
		digits /= 10
	}
	return string(out)
}
