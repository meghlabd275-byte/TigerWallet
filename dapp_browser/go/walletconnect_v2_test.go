package main

// walletconnect_v2_test.go — tests for the WalletConnect v2 cryptographic
// layer. These run against real primitives, not fixtures: the X25519 agreement
// is checked by deriving the same key from both sides, and the envelope tests
// check that tampering is detected rather than silently decrypted.

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/curve25519"
)

// TestX25519AgreementIsSymmetric checks that both peers reach the same
// symmetric key. This is the property the old implementation could not have:
// it returned the same random buffer as both the public and private key.
func TestX25519AgreementIsSymmetric(t *testing.T) {
	alice, err := GenerateX25519KeyPair()
	if err != nil {
		t.Fatalf("alice key pair: %v", err)
	}
	bob, err := GenerateX25519KeyPair()
	if err != nil {
		t.Fatalf("bob key pair: %v", err)
	}

	if alice.Public == alice.Private {
		t.Fatal("key pair has the same public and private key")
	}
	if alice.Public == bob.Public {
		t.Fatal("two independently generated key pairs share a public key")
	}

	aliceKey, err := DeriveSymKey(alice.Private, bob.Public)
	if err != nil {
		t.Fatalf("alice derives: %v", err)
	}
	bobKey, err := DeriveSymKey(bob.Private, alice.Public)
	if err != nil {
		t.Fatalf("bob derives: %v", err)
	}

	if string(aliceKey) != string(bobKey) {
		t.Fatal("the two peers derived different symmetric keys")
	}
	if len(aliceKey) != 32 {
		t.Fatalf("symmetric key is %d bytes, expected 32", len(aliceKey))
	}
}

// TestDeriveSymKeyRejectsLowOrderPoint checks that a peer cannot force a
// predictable shared secret by sending a small-order public key.
func TestDeriveSymKeyRejectsLowOrderPoint(t *testing.T) {
	our, err := GenerateX25519KeyPair()
	if err != nil {
		t.Fatalf("key pair: %v", err)
	}

	// All-zero and the identity point are the classic low-order choices.
	for _, lowOrder := range [][32]byte{
		{},
		{1},
	} {
		if _, err := DeriveSymKey(our.Private, lowOrder); err == nil {
			t.Fatalf("low-order public key %v was accepted", lowOrder)
		}
	}
}

// TestEnvelopeRoundTrip checks that a sealed envelope opens to the same bytes.
func TestEnvelopeRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("key: %v", err)
	}
	payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"eth_chainId"}`)

	envelope, err := SealEnvelope(key, payload, envelopeTypeMessage)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if strings.Contains(envelope, "eth_chainId") {
		t.Fatal("the plaintext is visible in the envelope")
	}

	got, msgType, err := OpenEnvelope(key, envelope)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("round trip changed the payload: %s", got)
	}
	if msgType != envelopeTypeMessage {
		t.Fatalf("envelope type is 0x%02x, expected 0x%02x", msgType, envelopeTypeMessage)
	}
}

// TestEnvelopeRejectsTampering checks that flipping any byte of the envelope
// makes it fail to open. ChaCha20-Poly1305 authenticates both the ciphertext
// and the header, so neither can be edited undetected.
func TestEnvelopeRejectsTampering(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	envelope, err := SealEnvelope(key, []byte(`{"method":"eth_sendTransaction"}`), envelopeTypeMessage)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(envelope)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	// The header type, the nonce and the ciphertext body are all covered.
	for _, offset := range []int{0, 2, len(raw) - 1} {
		mutated := append([]byte(nil), raw...)
		mutated[offset] ^= 0x01
		tampered := base64.RawURLEncoding.EncodeToString(mutated)
		if _, _, err := OpenEnvelope(key, tampered); err == nil {
			t.Fatalf("a tampered byte at offset %d was accepted", offset)
		}
	}
}

// TestEnvelopeRejectsWrongKey checks that a different symmetric key cannot open
// the envelope.
func TestEnvelopeRejectsWrongKey(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	other := make([]byte, 32)
	rand.Read(other)

	envelope, err := SealEnvelope(key, []byte("secret"), envelopeTypeMessage)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if _, _, err := OpenEnvelope(other, envelope); err == nil {
		t.Fatal("an envelope opened with the wrong key")
	}
}

// TestEnvelopeRejectsHeaderRewrite is the test that isolates header
// authentication. It swaps the type byte for another *valid* type while leaving
// the length and ciphertext untouched. Because the header is passed to the AEAD
// as additional data, decryption must fail; if the header were not
// authenticated, the envelope would open and be misrouted as a ping.
func TestEnvelopeRejectsHeaderRewrite(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	envelope, err := SealEnvelope(key, []byte(`{"method":"eth_sendTransaction"}`), envelopeTypeMessage)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(envelope)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	raw[0] = envelopeTypePing
	if _, _, err := OpenEnvelope(key, base64.RawURLEncoding.EncodeToString(raw)); err == nil {
		t.Fatal("an envelope whose type byte was rewritten opened successfully")
	}
}

// TestEnvelopeRejectsOversizedLength checks the header length cap, so a peer
// cannot force a large allocation by claiming a huge payload.
func TestEnvelopeRejectsOversizedLength(t *testing.T) {
	raw := make([]byte, headerSize+12+16)
	raw[0] = envelopeTypeMessage
	raw[1] = 0xff
	raw[2] = 0xff
	envelope := base64.RawURLEncoding.EncodeToString(raw)

	if _, _, err := OpenEnvelope(make([]byte, 32), envelope); err == nil {
		t.Fatal("an envelope claiming a 65535 byte payload was accepted")
	}
}

// TestEnvelopeRejectsUnknownType checks the type byte is validated.
func TestEnvelopeRejectsUnknownType(t *testing.T) {
	raw := make([]byte, headerSize+12+16)
	raw[0] = 0x7f
	envelope := base64.RawURLEncoding.EncodeToString(raw)

	if _, _, err := OpenEnvelope(make([]byte, 32), envelope); err == nil {
		t.Fatal("an envelope with an unknown type was accepted")
	}
}

// TestEnvelopeRejectsLengthMismatch checks that the declared length must match
// the decrypted payload, so a relay cannot make the receiver mis-slice it.
func TestEnvelopeRejectsLengthMismatch(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	payload := []byte("0123456789")

	envelope, err := SealEnvelope(key, payload, envelopeTypeMessage)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	raw, _ := base64.RawURLEncoding.DecodeString(envelope)
	// Rewrite the declared length while leaving the ciphertext alone. The
	// header is authenticated, so this must fail.
	raw[0] = envelopeTypeMessage
	raw[1] = 0x00
	raw[2] = byte(len(payload) - 1)

	if _, _, err := OpenEnvelope(key, base64.RawURLEncoding.EncodeToString(raw)); err == nil {
		t.Fatal("a rewritten length header was accepted")
	}
}

// TestSessionKeysDifferPerDirection checks that the send and receive keys are
// distinct, so a message cannot be reflected back as if the peer sent it.
func TestSessionKeysDifferPerDirection(t *testing.T) {
	sym := make([]byte, 32)
	rand.Read(sym)

	keys, err := deriveSessionKeys(sym)
	if err != nil {
		t.Fatalf("derive session keys: %v", err)
	}
	if string(keys.Send) == string(keys.Receive) {
		t.Fatal("send and receive keys are identical")
	}
	if string(keys.Send) == string(sym) || string(keys.Receive) == string(sym) {
		t.Fatal("a directional key equals the session key itself")
	}
}

// TestTopicIsStableForKey checks the topic derivation.
func TestTopicIsStableForKey(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	first := GenerateTopic(key)
	second := GenerateTopic(key)
	if first != second {
		t.Fatal("the same key produced two different topics")
	}
	if len(first) != 64 {
		t.Fatalf("topic is %d hex characters, expected 64", len(first))
	}

	other := make([]byte, 32)
	rand.Read(other)
	if GenerateTopic(other) == first {
		t.Fatal("two different keys produced the same topic")
	}
}

// TestCurve25519MatchesStdlib checks our agreement against the stdlib directly,
// which pins the HKDF step to a known input and output.
func TestCurve25519MatchesStdlib(t *testing.T) {
	priv := make([]byte, 32)
	rand.Read(priv)
	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		t.Fatalf("basepoint: %v", err)
	}

	peerPriv := make([]byte, 32)
	rand.Read(peerPriv)
	peerPub, err := curve25519.X25519(peerPriv, curve25519.Basepoint)
	if err != nil {
		t.Fatalf("peer basepoint: %v", err)
	}

	var ourPriv, peerPublic [32]byte
	copy(ourPriv[:], priv)
	copy(peerPublic[:], peerPub)

	ours, err := DeriveSymKey(ourPriv, peerPublic)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}

	shared, err := curve25519.X25519(priv, peerPub)
	if err != nil {
		t.Fatalf("stdlib agreement: %v", err)
	}
	expected, err := hkdfExpand(shared, nil, []byte("wc-v2-symkey"), 32)
	if err != nil {
		t.Fatalf("expected expansion: %v", err)
	}
	if string(ours) != string(expected) {
		t.Fatal("our agreement does not match the stdlib computation")
	}

	// And the peer must arrive at the same value.
	var theirPriv [32]byte
	copy(theirPriv[:], peerPriv)
	var ourPublic [32]byte
	copy(ourPublic[:], pub)
	theirs, err := DeriveSymKey(theirPriv, ourPublic)
	if err != nil {
		t.Fatalf("peer derive: %v", err)
	}
	if string(ours) != string(theirs) {
		t.Fatal("the two sides disagree")
	}
}

// TestReplayWindowRejectsDuplicate checks that the same envelope is accepted
// once and refused afterwards inside the window.
func TestReplayWindowRejectsDuplicate(t *testing.T) {
	window := newReplayWindow(300, 64)
	now := time.Now().Unix()

	if !window.Accept("envelope-one", now) {
		t.Fatal("the first delivery was refused")
	}
	if window.Accept("envelope-one", now+1) {
		t.Fatal("a duplicate delivery was accepted")
	}
	if !window.Accept("envelope-two", now+1) {
		t.Fatal("a distinct envelope was refused")
	}
	// After the window expires the digest is forgotten, which is correct: the
	// envelope itself would have expired at the relay too.
	if !window.Accept("envelope-one", now+400) {
		t.Fatal("an envelope was refused after the window expired")
	}
}

// TestReplayWindowBoundsMemory checks the cap holds under a burst.
func TestReplayWindowBoundsMemory(t *testing.T) {
	window := newReplayWindow(300, 8)
	now := time.Now().Unix()

	for i := 0; i < 50; i++ {
		if !window.Accept(fmt.Sprintf("envelope-%d", i), now) {
			t.Fatalf("envelope %d was refused inside the cap", i)
		}
	}
	window.mu.Lock()
	size := len(window.seen)
	window.mu.Unlock()
	if size > 8 {
		t.Fatalf("the replay window grew to %d entries, above its cap of 8", size)
	}
}

// TestVerifyRelayAuthJWT checks a correctly signed token is accepted and that
// the algorithm, audience and expiry are all enforced.
func TestVerifyRelayAuthJWT(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	original := RelayAuthKey
	RelayAuthKey = pub
	defer func() { RelayAuthKey = original }()

	now := time.Now().Unix()
	claims := RelayAuthJWT{
		Issuer:   "did:key:z6Mk",
		Subject:  "test",
		Audience: "walletconnect",
		IssuedAt: now - 10,
		Expiry:   now + 600,
	}
	token := signTestJWT(t, priv, "EdDSA", claims)

	parsed, err := VerifyRelayAuthJWT(token, "walletconnect", now)
	if err != nil {
		t.Fatalf("a valid token was rejected: %v", err)
	}
	if parsed.Audience != "walletconnect" {
		t.Fatalf("audience parsed as %q", parsed.Audience)
	}

	if _, err := VerifyRelayAuthJWT(token, "someone-else", now); err == nil {
		t.Fatal("a token for another audience was accepted")
	}
	if _, err := VerifyRelayAuthJWT(token, "walletconnect", now+700); err == nil {
		t.Fatal("an expired token was accepted")
	}
}

// TestVerifyRelayAuthJWTRejectsAlgNone checks that the classic "alg: none"
// substitution is refused: accepting it would let anyone mint a token.
func TestVerifyRelayAuthJWTRejectsAlgNone(t *testing.T) {
	now := time.Now().Unix()
	claims := RelayAuthJWT{Audience: "walletconnect", Expiry: now + 600}

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := header + "." + payload + "."

	if _, err := VerifyRelayAuthJWT(token, "walletconnect", now); err == nil {
		t.Fatal("a token with alg=none was accepted")
	}
}

// TestVerifyRelayAuthJWTRejectsWrongAlgorithm isolates the algorithm check. The
// signature here is a genuine Ed25519 signature, so only the `alg` claim is
// wrong. The classic attack is to relabel a token "HS256" and have a naive
// verifier treat the relay's public key as an HMAC secret; rejecting anything
// that is not EdDSA closes that off before the signature is even considered.
func TestVerifyRelayAuthJWTRejectsWrongAlgorithm(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	original := RelayAuthKey
	RelayAuthKey = pub
	defer func() { RelayAuthKey = original }()

	now := time.Now().Unix()
	claims := RelayAuthJWT{Audience: "walletconnect", Expiry: now + 600}

	// A correctly signed token, but labelled with an HMAC algorithm.
	token := signTestJWT(t, priv, "HS256", claims)

	if _, err := VerifyRelayAuthJWT(token, "walletconnect", now); err == nil {
		t.Fatal("a token labelled HS256 was accepted despite a valid Ed25519 signature")
	}

	// The same claims signed and labelled correctly must still be accepted, so
	// the rejection above is caused by the algorithm and nothing else.
	good := signTestJWT(t, priv, "EdDSA", claims)
	if _, err := VerifyRelayAuthJWT(good, "walletconnect", now); err != nil {
		t.Fatalf("the correctly labelled token was rejected: %v", err)
	}
}

// TestVerifyRelayAuthJWTRejectsBadSignature checks that a modified payload
// fails verification.
func TestVerifyRelayAuthJWTRejectsBadSignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	original := RelayAuthKey
	RelayAuthKey = pub
	defer func() { RelayAuthKey = original }()

	now := time.Now().Unix()
	token := signTestJWT(t, priv, "EdDSA", RelayAuthJWT{Audience: "walletconnect", Expiry: now + 600})

	// Swap the payload for one that extends the expiry.
	parts := strings.Split(token, ".")
	payloadBytes, _ := json.Marshal(RelayAuthJWT{Audience: "walletconnect", Expiry: now + 100000})
	forged := parts[0] + "." + base64.RawURLEncoding.EncodeToString(payloadBytes) + "." + parts[2]

	if _, err := VerifyRelayAuthJWT(forged, "walletconnect", now); err == nil {
		t.Fatal("a token with a forged payload was accepted")
	}
}

// TestVerifyRelayAuthJWTRejectsMalformed checks the shape checks.
func TestVerifyRelayAuthJWTRejectsMalformed(t *testing.T) {
	now := time.Now().Unix()
	for _, token := range []string{"", "a", "a.b", "a.b.c.d", "!!!.!!!.!!!"} {
		if _, err := VerifyRelayAuthJWT(token, "", now); err == nil {
			t.Fatalf("malformed token %q was accepted", token)
		}
	}
}

// TestParsePairingURIRoundTrip checks a URI this service builds parses back to
// the same fields.
func TestParsePairingURIRoundTrip(t *testing.T) {
	uri := BuildPairingURI("topic123", "pubkey456", "wss://relay.walletconnect.org", 1700000000, []string{"eth_sendTransaction"})

	parsed, err := ParsePairingURI(uri)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Topic != "topic123" {
		t.Fatalf("topic parsed as %q", parsed.Topic)
	}
	if parsed.PublicKey != "pubkey456" {
		t.Fatalf("symKey parsed as %q", parsed.PublicKey)
	}
	if parsed.RelayURL != "wss://relay.walletconnect.org" {
		t.Fatalf("relay parsed as %q", parsed.RelayURL)
	}
	if parsed.Expiry != 1700000000 {
		t.Fatalf("expiry parsed as %d", parsed.Expiry)
	}
	if len(parsed.Methods) != 1 || parsed.Methods[0] != "eth_sendTransaction" {
		t.Fatalf("methods parsed as %v", parsed.Methods)
	}
}

// TestParsePairingURIRejectsBadInput checks that malformed URIs are refused
// rather than partially accepted, since every field reaches a remote peer.
func TestParsePairingURIRejectsBadInput(t *testing.T) {
	cases := []string{
		"",
		"https://example.com",
		"wc:topic123",                      // no query string
		"wc:topic123@1?symKey=abc",         // wrong version
		"wc:@2?symKey=abc",                 // empty topic
		"wc:topic123@2?relay-protocol=irn", // no symKey
		"wc:topic123@2?symKey=",            // empty symKey
	}
	for _, uri := range cases {
		if _, err := ParsePairingURI(uri); err == nil {
			t.Fatalf("malformed URI %q was accepted", uri)
		}
	}
}

// TestGenerateX25519KeyPairIsRandom checks two pairs never coincide.
func TestGenerateX25519KeyPairIsRandom(t *testing.T) {
	seen := make(map[[32]byte]bool)
	for i := 0; i < 64; i++ {
		pair, err := GenerateX25519KeyPair()
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if seen[pair.Public] {
			t.Fatal("a public key repeated across generations")
		}
		seen[pair.Public] = true
	}
}

// signTestJWT builds a token with the given algorithm and claims.
func signTestJWT(t *testing.T, priv ed25519.PrivateKey, alg string, claims RelayAuthJWT) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"alg":%q,"typ":"JWT"}`, alg)))
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signature := ed25519.Sign(priv, jwtSigningInput(header, payload))
	return header + "." + payload + "." + base64.RawURLEncoding.EncodeToString(signature)
}
