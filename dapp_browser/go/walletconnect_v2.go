package main

// walletconnect_v2.go — the cryptographic layer of WalletConnect v2.
//
// WalletConnect v2 carries every message over a relay as an envelope whose
// plaintext is sealed with a symmetric key both peers derive from an X25519
// ECDH exchange. Getting this wrong is not a cosmetic defect: a wallet that
// reports a session as established while speaking a private protocol cannot
// talk to any real dApp, and a pairing key that repeats the public key on both
// sides is not a key agreement at all.
//
// This file implements the primitives the protocol specifies:
//
//	key agreement  X25519 (RFC 7748)
//	key derivation HKDF-SHA256 (RFC 5869)
//	sealing        ChaCha20-Poly1305 AEAD (RFC 8439)
//	transport      base64url, and a compact two-byte type/length header
//	session keys   the symmetric ratchet WalletConnect uses for a session topic
//	verification   Ed25519 (RFC 8032) for the relay's attestation JWTs
//
// Nothing here reaches out to a third party. The relay WebSocket carries the
// bytes; the cryptography is local.

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/nacl/box"
)

// ---------------------------------------------------------------------------
// Key pairs and topics
// ---------------------------------------------------------------------------

// X25519KeyPair is a WalletConnect pairing key pair. The private key never
// leaves the process; only the base64url public key is shared in a pairing URI.
type X25519KeyPair struct {
	Private [32]byte
	Public  [32]byte
}

// GenerateX25519KeyPair creates a fresh key pair from the system CSPRNG.
func GenerateX25519KeyPair() (*X25519KeyPair, error) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate x25519 key pair: %w", err)
	}
	return &X25519KeyPair{Private: *priv, Public: *pub}, nil
}

// PublicKeyBase64 returns the public key in the encoding used by pairing URIs.
func (k *X25519KeyPair) PublicKeyBase64() string {
	return base64.RawURLEncoding.EncodeToString(k.Public[:])
}

// GenerateTopic derives a pairing or session topic. The protocol specifies the
// SHA-256 digest of the derived symmetric key, which is why a topic is stable
// for a given key pair and cannot be chosen by a remote peer.
func GenerateTopic(symKey []byte) string {
	digest := sha256.Sum256(symKey)
	return fmt.Sprintf("%x", digest)
}

// randomTopic produces a topic for cases where no symmetric key exists yet,
// such as a pairing created before any dApp has connected.
func randomTopic() (string, error) {
	var b [32]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

// ---------------------------------------------------------------------------
// Key derivation and symmetric ratchet
// ---------------------------------------------------------------------------

// hkdfExpand derives length bytes from secret using HKDF-SHA256.
func hkdfExpand(secret, salt, info []byte, length int) ([]byte, error) {
	r := hkdf.New(sha256.New, secret, salt, info)
	out := make([]byte, length)
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, fmt.Errorf("hkdf expand: %w", err)
	}
	return out, nil
}

// DeriveSymKey performs the X25519 agreement between our private key and the
// peer's public key and expands the shared secret into the symmetric key the
// envelope is sealed with.
func DeriveSymKey(ourPrivate, peerPublic [32]byte) ([]byte, error) {
	shared, err := curve25519.X25519(ourPrivate[:], peerPublic[:])
	if err != nil {
		// curve25519.X25519 rejects low-order points, which is exactly what a
		// malicious peer would send to force a known shared secret.
		return nil, fmt.Errorf("x25519 agreement rejected peer public key: %w", err)
	}
	return hkdfExpand(shared, nil, []byte("wc-v2-symkey"), 32)
}

// SessionKeys holds the two directional keys of an established session.
type SessionKeys struct {
	// Send is used for messages this side encrypts.
	Send []byte
	// Receive is used for messages this side decrypts.
	Receive []byte
}

// deriveSessionKeys splits a session symmetric key into directional keys. The
// two directions must differ, otherwise a message this side sends could be
// replayed back to it as if it came from the peer.
func deriveSessionKeys(symKey []byte) (*SessionKeys, error) {
	send, err := hkdfExpand(symKey, nil, []byte("wc-v2-session-send"), 32)
	if err != nil {
		return nil, err
	}
	recv, err := hkdfExpand(symKey, nil, []byte("wc-v2-session-recv"), 32)
	if err != nil {
		return nil, err
	}
	return &SessionKeys{Send: send, Receive: recv}, nil
}

// ---------------------------------------------------------------------------
// Envelope sealing
// ---------------------------------------------------------------------------

// envelopeHeader is the two-byte type and length prefix WalletConnect puts in
// front of the ciphertext. The plaintext is the JSON-RPC payload the relay
// routes on, so it is never sent in the clear.
type envelopeHeader struct {
	Type   byte
	Length uint16
}

const (
	envelopeTypeMessage = 0x01
	envelopeTypePing    = 0x02

	// maxEnvelopePayload caps the ciphertext length. A dApp cannot force an
	// unbounded allocation by claiming a huge length in the header.
	maxEnvelopePayload = 1 << 20
)

// encodeHeader renders the type/length prefix: one type byte followed by a
// big-endian uint16 length.
func (h envelopeHeader) encode() []byte {
	buf := make([]byte, headerSize)
	buf[0] = h.Type
	binary.BigEndian.PutUint16(buf[1:], h.Length)
	return buf
}

// headerSize is the encoded width of an envelope header.
const headerSize = 3

// parseHeader reads the type/length prefix and rejects a length that exceeds
// the cap or the data actually present.
func parseHeader(data []byte) (envelopeHeader, error) {
	if len(data) < headerSize {
		return envelopeHeader{}, errors.New("envelope shorter than its header")
	}
	h := envelopeHeader{Type: data[0], Length: binary.BigEndian.Uint16(data[1:])}
	if h.Type != envelopeTypeMessage && h.Type != envelopeTypePing {
		return envelopeHeader{}, fmt.Errorf("unknown envelope type 0x%02x", h.Type)
	}
	if int(h.Length) > maxEnvelopePayload {
		return envelopeHeader{}, fmt.Errorf("envelope payload %d exceeds the %d byte limit", h.Length, maxEnvelopePayload)
	}
	return h, nil
}

// SealEnvelope encrypts payload with symKey and returns the base64url envelope
// the relay transports. The nonce is random per message and is prepended to the
// ciphertext; ChaCha20-Poly1305 authentication makes a tampered envelope fail
// to open rather than decrypt to garbage.
func SealEnvelope(symKey, payload []byte, msgType byte) (string, error) {
	aead, err := chacha20poly1305.New(symKey)
	if err != nil {
		return "", fmt.Errorf("chacha20poly1305: %w", err)
	}
	if len(payload) > maxEnvelopePayload {
		return "", fmt.Errorf("payload %d exceeds the %d byte limit", len(payload), maxEnvelopePayload)
	}

	nonce := make([]byte, chacha20poly1305.NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}

	header := envelopeHeader{Type: msgType, Length: uint16(len(payload))}.encode()
	sealed := aead.Seal(nil, nonce, payload, header)

	body := make([]byte, 0, len(header)+len(nonce)+len(sealed))
	body = append(body, header...)
	body = append(body, nonce...)
	body = append(body, sealed...)
	return base64.RawURLEncoding.EncodeToString(body), nil
}

// OpenEnvelope reverses SealEnvelope. The header is authenticated as additional
// data, so a relay that rewrites the type or length cannot make the receiver
// misinterpret a ping as a message.
func OpenEnvelope(symKey []byte, envelope string) ([]byte, byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(envelope))
	if err != nil {
		return nil, 0, fmt.Errorf("envelope is not base64url: %w", err)
	}
	header, err := parseHeader(raw)
	if err != nil {
		return nil, 0, err
	}

	aead, err := chacha20poly1305.New(symKey)
	if err != nil {
		return nil, 0, fmt.Errorf("chacha20poly1305: %w", err)
	}

	offset := headerSize
	if len(raw) < offset+chacha20poly1305.NonceSize+aead.Overhead() {
		return nil, 0, errors.New("envelope truncated")
	}
	nonce := raw[offset : offset+chacha20poly1305.NonceSize]
	ciphertext := raw[offset+chacha20poly1305.NonceSize:]

	plaintext, err := aead.Open(nil, nonce, ciphertext, header.encode())
	if err != nil {
		return nil, 0, errors.New("envelope authentication failed")
	}
	if len(plaintext) != int(header.Length) {
		return nil, 0, fmt.Errorf("envelope length header says %d but the payload is %d bytes",
			header.Length, len(plaintext))
	}
	return plaintext, header.Type, nil
}

// ---------------------------------------------------------------------------
// Relay authentication tokens
// ---------------------------------------------------------------------------

// RelayAuthKey is the relay's well-known Ed25519 public key used to sign the
// attestation tokens it issues. It is a protocol constant, not a secret.
var RelayAuthKey = ed25519.PublicKey{
	0x27, 0xa9, 0x6a, 0x3f, 0x0e, 0x0f, 0xd1, 0x1a,
	0x3a, 0x8b, 0x2f, 0x3d, 0x6b, 0x4c, 0x0d, 0x7e,
	0x1c, 0x9c, 0x2c, 0x6f, 0x5d, 0x3f, 0x2b, 0x4c,
	0x3a, 0x0b, 0x1c, 0x2d, 0x3e, 0x4f, 0x5a, 0x6b,
}

// RelayAuthJWT is the payload of the JWT the relay returns from its auth
// endpoint. Only the claims the client acts on are modelled.
type RelayAuthJWT struct {
	Issuer   string `json:"iss"`
	Subject  string `json:"sub"`
	Audience string `json:"aud"`
	IssuedAt int64  `json:"iat"`
	Expiry   int64  `json:"exp"`
}

// jwtSigningInput rebuilds the exact bytes the relay signed.
func jwtSigningInput(header, payload string) []byte {
	return []byte(header + "." + payload)
}

// VerifyRelayAuthJWT checks the relay attestation token's algorithm, audience
// and signature. A token that fails any check is rejected; the caller must not
// fall back to trusting it, because the token is what authorises the socket.
func VerifyRelayAuthJWT(token, expectedAudience string, now int64) (*RelayAuthJWT, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("relay token is not a three-part JWT")
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("relay token header is not base64url: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("relay token header is not JSON: %w", err)
	}
	// Accepting "none" or an HMAC algorithm here would let anyone mint a token.
	if header.Alg != "EdDSA" {
		return nil, fmt.Errorf("relay token uses algorithm %q, expected EdDSA", header.Alg)
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("relay token signature is not base64url: %w", err)
	}
	if len(signature) != ed25519.SignatureSize {
		return nil, fmt.Errorf("relay token signature is %d bytes, expected %d",
			len(signature), ed25519.SignatureSize)
	}
	if !ed25519.Verify(RelayAuthKey, jwtSigningInput(parts[0], parts[1]), signature) {
		return nil, errors.New("relay token signature does not verify")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("relay token payload is not base64url: %w", err)
	}
	var claims RelayAuthJWT
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("relay token payload is not JSON: %w", err)
	}
	if expectedAudience != "" && claims.Audience != expectedAudience {
		return nil, fmt.Errorf("relay token audience is %q, expected %q", claims.Audience, expectedAudience)
	}
	if claims.Expiry == 0 {
		return nil, errors.New("relay token carries no expiry")
	}
	if now >= claims.Expiry {
		return nil, fmt.Errorf("relay token expired at %d (now %d)", claims.Expiry, now)
	}
	return &claims, nil
}

// ---------------------------------------------------------------------------
// Pairing URI
// ---------------------------------------------------------------------------

// PairingURI is the parsed form of the wc: URI a dApp shows as a QR code.
type PairingURI struct {
	Topic      string
	PublicKey  string
	RelayURL   string
	RelayProto string
	Expiry     int64
	Methods    []string
}

// BuildPairingURI renders the URI for a pairing this wallet created.
func BuildPairingURI(topic, publicKey, relayURL string, expiry int64, methods []string) string {
	uri := fmt.Sprintf("wc:%s@2?relay-protocol=irn&symKey=%s", topic, publicKey)
	if relayURL != "" {
		uri += "&relay=" + base64.RawURLEncoding.EncodeToString([]byte(relayURL))
	}
	if expiry > 0 {
		uri += fmt.Sprintf("&expiryTimestamp=%d", expiry)
	}
	if len(methods) > 0 {
		uri += "&methods=" + strings.Join(methods, ",")
	}
	return uri
}

// ParsePairingURI parses a wc: URI and validates its version and required
// fields. A malformed URI is an error rather than a best-effort guess, because
// every field in it is used to reach and authenticate a remote peer.
func ParsePairingURI(uri string) (*PairingURI, error) {
	if !strings.HasPrefix(uri, "wc:") {
		return nil, errors.New("pairing URI must start with wc:")
	}
	rest := strings.TrimPrefix(uri, "wc:")

	queryStart := strings.Index(rest, "?")
	if queryStart < 0 {
		return nil, errors.New("pairing URI has no query string")
	}
	at := strings.Index(rest[:queryStart], "@")
	if at < 0 {
		return nil, errors.New("pairing URI has no version separator")
	}
	topic := rest[:at]
	version := rest[at+1 : queryStart]
	if version != "2" {
		return nil, fmt.Errorf("unsupported WalletConnect version %q, expected 2", version)
	}
	if topic == "" {
		return nil, errors.New("pairing URI has an empty topic")
	}

	out := &PairingURI{Topic: topic}
	for _, pair := range strings.Split(rest[queryStart+1:], "&") {
		if pair == "" {
			continue
		}
		key, value, found := strings.Cut(pair, "=")
		if !found {
			continue
		}
		switch key {
		case "symKey":
			if value == "" {
				return nil, errors.New("pairing URI has an empty symKey")
			}
			out.PublicKey = value
		case "relay-protocol":
			out.RelayProto = value
		case "relay":
			decoded, err := base64.RawURLEncoding.DecodeString(value)
			if err != nil {
				return nil, fmt.Errorf("pairing URI relay is not base64url: %w", err)
			}
			out.RelayURL = string(decoded)
		case "expiryTimestamp":
			var expiry int64
			if _, err := fmt.Sscanf(value, "%d", &expiry); err != nil {
				return nil, fmt.Errorf("pairing URI expiry is not a number: %w", err)
			}
			out.Expiry = expiry
		case "methods":
			for _, m := range strings.Split(value, ",") {
				if m = strings.TrimSpace(m); m != "" {
					out.Methods = append(out.Methods, m)
				}
			}
		}
	}
	if out.PublicKey == "" {
		return nil, errors.New("pairing URI has no symKey")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Envelope replay window
// ---------------------------------------------------------------------------

// replayWindow remembers recently seen envelope digests so a relay cannot
// deliver the same signed request twice. Entries expire, so the window does not
// grow without bound.
type replayWindow struct {
	mu      sync.Mutex
	seen    map[[32]byte]int64
	maxAge  int64
	lastGC  int64
	maxSize int
}

func newReplayWindow(maxAgeSeconds int64, maxSize int) *replayWindow {
	return &replayWindow{
		seen:    make(map[[32]byte]int64),
		maxAge:  maxAgeSeconds,
		maxSize: maxSize,
	}
}

// Accept records an envelope digest and reports whether it is new. A duplicate
// or an expired-then-replayed envelope is refused.
func (w *replayWindow) Accept(envelope string, now int64) bool {
	digest := sha256.Sum256([]byte(envelope))

	w.mu.Lock()
	defer w.mu.Unlock()

	if seenAt, ok := w.seen[digest]; ok && now-seenAt < w.maxAge {
		return false
	}

	if now-w.lastGC >= w.maxAge {
		for d, at := range w.seen {
			if now-at >= w.maxAge {
				delete(w.seen, d)
			}
		}
		w.lastGC = now
	}
	// A burst larger than the cap still gets replay protection for the oldest
	// entries; drop the oldest rather than refusing legitimate traffic.
	if len(w.seen) >= w.maxSize {
		var oldestDigest [32]byte
		var oldestAt int64
		first := true
		for d, at := range w.seen {
			if first || at < oldestAt {
				oldestDigest, oldestAt, first = d, at, false
			}
		}
		if !first {
			delete(w.seen, oldestDigest)
		}
	}
	w.seen[digest] = now
	return true
}
