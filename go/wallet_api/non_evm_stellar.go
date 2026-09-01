package main

// non_evm_stellar.go — Stellar + Pi Network SDK (Pi runs a stellar-core
// fork): strkey 'G' addresses, Ed25519 transaction signing, broadcast via
// Horizon (/transactions). Pi passphrase is env-configurable
// (PI_NETWORK_PASSPHRASE, default "Pi Network"); Stellar mainnet uses its
// fixed passphrase. The built Transaction is the legacy-v1 layout, still
// accepted by current Horizons.

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	ed "golang.org/x/crypto/ed25519"
)

// crc16XModem computes the CRC16-XModem (poly 0x1021, init 0x0000).
func crc16XModem(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// stellarNetworkPass resolves the network passphrase per chain type.
func stellarNetworkPass(chainType string) string {
	if chainType == "pi" {
		if env := strings.TrimSpace(os.Getenv("PI_NETWORK_PASSPHRASE")); env != "" {
			return env
		}
		return "Pi Network"
	}
	return "Public Global Stellar Network ; September 2015"
}

// StrKeyAddress computes the base32 'G' strkey for an ed25519 pubkey.
func StrKeyAddress(seed []byte, path string) (string, error) {
	pub, err := edPubKey(seed, path)
	if err != nil {
		return "", err
	}
	return strkeyEncode(6, pub[:32]) // version byte 0x06 = 'G' (account id)
}

// strkeyEncode encodes a strkey: version (shifted <<3 per SEP-0023), data,
// CRC16-XModem (LE 2B), RFC-4648 uppercase base32 without padding.
func strkeyEncode(version byte, data []byte) (string, error) {
	raw := append([]byte{version << 3}, data...)
	crc := crc16XModem(raw)
	raw = append(raw, byte(crc), byte(crc>>8))
	return base32LowerEnc(raw), nil
}

// base32LowerEnc uses the RFC-4648 uppercase alphabet (strkey canonical form)
// with no padding; the name is historical.
func base32LowerEnc(b []byte) string {
	const alpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	var out strings.Builder
	var acc uint16
	bits := 0
	for _, x := range b {
		acc = (acc << 8) | uint16(x)
		bits += 8
		for bits >= 5 {
			bits -= 5
			out.WriteByte(alpha[(acc>>bits)&31])
		}
	}
	if bits > 0 {
		out.WriteByte(alpha[(acc<<(5-bits))&31])
	}
	return out.String()
}

func base32LowerDec(s string) ([]byte, error) {
	const alpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	var acc uint32
	bits := 0
	var out []byte
	for _, c := range []byte(strings.ToUpper(s)) {
		idx := strings.IndexByte(alpha, c)
		if idx < 0 {
			return nil, fmt.Errorf("strkey base32: invalid char %q", c)
		}
		acc = (acc << 5) | uint32(idx)
		bits += 5
		for bits >= 8 {
			bits -= 8
			out = append(out, byte((acc>>bits)&0xff))
		}
	}
	return out, nil
}

// strkeyDecodePub decodes a G-address into the 32-byte public key.
func strkeyDecodePub(addr string) ([]byte, error) {
	if len(addr) != 56 || addr[0] != 'G' {
		return nil, errors.New("invalid stellar/pi address")
	}
	raw, err := base32LowerDec(addr)
	if err != nil {
		return nil, err
	}
	if len(raw) < 35 {
		return nil, errors.New("short stellar address payload")
	}
	if raw[0] != 6 {
		return nil, errors.New("address is not a stellar account (G) address")
	}
	return raw[1:33], nil
}

// ---------- XDR encode helpers (big-endian fixed) ----------

type xdr struct{ b []byte }

func (x *xdr) u32(v uint32) { x.b = append(x.b, byte(v>>24), byte(v>>16), byte(v>>8), byte(v)) }
func (x *xdr) u64(v uint64) { x.u32(uint32(v >> 32)); x.u32(uint32(v)) }
func (x *xdr) i64(v int64)  { x.u64(uint64(v)) }
func (x *xdr) raw(b []byte) { x.b = append(x.b, b...) }

// StellarBuildSend builds a payment envelope: the Transaction structure
// (legacy v1) and one DecoratedSignature (hint = last 4 pubkey bytes).
func StellarBuildSend(ctx context.Context, seed []byte, path, horizon, chainType, to string, amountStrop int64, sequenceNum uint64, fee uint32, broadcast bool) (string, string, error) {
	priv, pub, err := edKeypair(seed, path)
	if err != nil {
		return "", "", err
	}
	toDecoded, err := strkeyDecodePub(to)
	if err != nil {
		return "", "", err
	}

	tx := &xdr{}
	tx.u32(0)                  // PublicKey variant: ED25519
	tx.raw(pub[0:32])          // source account
	tx.u32(fee)
	tx.u64(sequenceNum)
	tx.u32(0)                  // timeBounds absent (legacy)
	tx.u32(0)                  // MEMO_NONE
	tx.u32(1)                  // operations vec: 1
	tx.u32(0)                  // op.sourceAccount absent
	tx.u32(2)                  // op body: PAYMENT
	tx.u32(0)                  // dest MuxedAccount variant: ED25519
	tx.raw(toDecoded)          // dest
	tx.u32(0)                  // ASSET_TYPE_NATIVE
	tx.i64(amountStrop)
	tx.u32(0)                  // TransactionExt variant 0
	txB := tx.b

	network := stellarNetworkPass(chainType)
	pathSum := sha256.Sum256([]byte(network))
	hashSrc := append(pathSum[:], txB...)
	hash := sha256.Sum256(hashSrc)

	sig := ed.Sign(priv, hash[:])
	hint := pub[len(pub)-4:]

	sigs := &xdr{}
	sigs.u32(1)                // one decorated signature
	sigs.raw(hint)
	sigs.u32(uint32(len(sig)))
	sigs.raw(sig)

	env := &xdr{}
	env.u32(2)                 // ENVELOPE_TYPE_TX
	env.raw(txB)
	env.raw(sigs.b)
	xdrB := env.b

	b64 := base64.StdEncoding.EncodeToString(xdrB)
	if !broadcast {
		return b64, "", nil
	}
	form := fmt.Sprintf("tx=%s", b64)
	url := strings.TrimRight(horizon, "/") + "/transactions"
	raw, err := postFormURLEncoded(ctx, url, form)
	if err != nil {
		return "", "", fmt.Errorf("horizon broadcast: %w", err)
	}
	var parsed struct {
		Hash string `json:"hash"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || parsed.Hash == "" {
		return "", "", errors.New("horizon broadcast: malformed reply")
	}
	return b64, parsed.Hash, nil
}
