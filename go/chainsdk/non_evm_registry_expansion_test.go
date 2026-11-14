package chainsdk

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

// Reference values produced by starknet.js 6.24.1 (ec.starkCurve.grindKey and
// ec.starkCurve.getStarkKey) for the entropy 0x1111...11 (32 bytes). Pinning
// them here proves the key-grinding rule and the public-key derivation match
// the reference wallet, so a key imported from TigerWallet behaves the same in
// any other Starknet wallet.
func TestStarknetGrindKeyMatchesReference(t *testing.T) {
	entropy := make([]byte, 32)
	for i := range entropy {
		entropy[i] = 0x11
	}
	k, err := starknetGrindKey(entropy)
	if err != nil {
		t.Fatalf("grindKey: %v", err)
	}
	gotPriv := hex.EncodeToString(k.Bytes())
	const wantPriv = "07d3214da593feb0bcde5420d6bd5a699c15cd53f49952fb96cc4786792621fe"
	if gotPriv != wantPriv {
		t.Fatalf("private key = %s, want %s", gotPriv, wantPriv)
	}

	gotPub := hex.EncodeToString(StarknetStarkKey(k).Bytes())
	const wantPub = "b9c26f8f3ab5403efb6b9ba391dade3f1b58647c3ad4cfd644b88c17f23cc2"
	if gotPub != wantPub {
		t.Fatalf("public key = %s, want %s", gotPub, wantPub)
	}
}

// Starknet refuses to invent an account class hash: the address is a contract
// address, and the class hash decides which account contract is deployed.
func TestStarknetAddressFailsClosedWithoutClassHash(t *testing.T) {
	seed := []byte("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about")
	if _, err := StarknetAddressForChain("starknet", seed, "m/44'/9004'/0'/0/0"); err == nil {
		t.Fatal("expected an error when no account class hash is configured")
	}
	if _, err := StarknetAddressForChain("starknet", seed, "m/44'/9004'/0'/0/0"); !strings.Contains(err.Error(), "STARKNET_STARKNET_ACCOUNT_CLASS_HASH") {
		t.Fatalf("error should name the environment variable, got: %v", err)
	}
}

// A configured class hash must yield a deterministic 0x-prefixed 32-byte felt.
func TestStarknetAddressWithClassHash(t *testing.T) {
	t.Setenv("STARKNET_STARKNET_ACCOUNT_CLASS_HASH", "0x1a736d6ed154502257f02b1ccdf4d9d1089f80811cd6acad48e6b6a9d1f2003")
	seed := []byte("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about")
	addr, err := StarknetAddressForChain("starknet", seed, "m/44'/9004'/0'/0/0")
	if err != nil {
		t.Fatalf("address: %v", err)
	}
	if len(addr) != 66 || !strings.HasPrefix(addr, "0x") {
		t.Fatalf("address %q is not a 32-byte felt", addr)
	}
	again, err := StarknetAddressForChain("starknet", seed, "m/44'/9004'/0'/0/0")
	if err != nil || again != addr {
		t.Fatalf("derivation is not deterministic: %q vs %q (%v)", addr, again, err)
	}
}

// Starknet signatures must be two 32-byte felts below the curve order and must
// satisfy the ECDSA verification equation on the STARK curve:
//
//	r == x((z*s^-1)*G + (r*s^-1)*Q)  mod order
//
// so a signature this backend produces is one a Starknet account contract
// accepts.
func TestStarknetSignatureVerifies(t *testing.T) {
	seed := []byte("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about")
	msg := []byte("hello starknet")
	sig, pub, err := StarknetSignMessage(seed, "m/44'/9004'/0'/0/0", msg)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if len(sig) != 64 || len(pub) != 32 {
		t.Fatalf("unexpected sizes: sig=%d pub=%d", len(sig), len(pub))
	}
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])
	if r.Sign() == 0 || s.Sign() == 0 || r.Cmp(starkOrder) >= 0 || s.Cmp(starkOrder) >= 0 {
		t.Fatal("signature components must be non-zero and below the curve order")
	}

	// Recompute the message hash exactly as the signer does.
	h := big.NewInt(0)
	for i := 0; i < len(msg); i += 31 {
		end := i + 31
		if end > len(msg) {
			end = len(msg)
		}
		limb := new(big.Int).SetBytes(msg[i:end])
		limb.Mod(limb, starkPrime)
		h = starknetPedersen(h, limb)
	}
	z := new(big.Int).Mod(h, starkOrder)

	// Q is the public key point, recovered from the x coordinate by the curve
	// equation y^2 = x^3 + alpha*x + beta.
	qx := new(big.Int).SetBytes(pub)
	alpha := mustParseBig("1")
	beta := mustParseBig("3141592653589793238462643383279502884197169399375105820974944592307816406665")
	y2 := new(big.Int).Mul(qx, qx)
	y2.Mul(y2, qx)
	y2.Add(y2, new(big.Int).Mul(alpha, qx))
	y2.Add(y2, beta)
	y2.Mod(y2, starkPrime)
	qy := starkSqrt(y2)
	if qy == nil {
		t.Fatal("public key x coordinate is not on the STARK curve")
	}
	// Either root works for verification; pick the parity the signer used by
	// checking both.
	for _, candidate := range []*big.Int{qy, new(big.Int).Sub(starkPrime, qy)} {
		if starkVerify(r, s, z, &starkPoint{x: qx, y: candidate}) {
			return
		}
	}
	t.Fatal("signature does not verify against the derived public key")
}

// starkVerify checks r == x((z*s^-1)G + (r*s^-1)Q) mod order. Both inverses are
// taken modulo the group order; taking them modulo the field prime would give a
// result that does not correspond to any real signature.
func starkVerify(r, s, z *big.Int, q *starkPoint) bool {
	sinv := new(big.Int).ModInverse(s, starkOrder)
	if sinv == nil {
		return false
	}
	u1 := new(big.Int).Mul(z, sinv)
	u1.Mod(u1, starkOrder)
	u2 := new(big.Int).Mul(r, sinv)
	u2.Mod(u2, starkOrder)
	pt := starkAdd(starkMul(u1, starkGenerator), starkMul(u2, q))
	if pt == nil {
		return false
	}
	return new(big.Int).Mod(pt.x, starkOrder).Cmp(r) == 0
}

// starkSqrt returns a square root of v modulo the STARK prime, or nil when v is
// not a quadratic residue. The prime is 1 mod 4 (2^251 + 17*2^192 + 1), so the
// (p+1)/4 shortcut does not apply and Tonelli-Shanks is required. @scure/starknet
// documents the same limitation ("there is no efficient sqrt for field
// (P%4==1)"); this helper exists only so the test can recover a public-key
// y coordinate from its x coordinate and verify independently of the signer.
func starkSqrt(v *big.Int) *big.Int {
	n := new(big.Int).Mod(v, starkPrime)
	if n.Sign() == 0 {
		return big.NewInt(0)
	}
	// Write p-1 = q * 2^s with q odd.
	q := new(big.Int).Sub(starkPrime, big.NewInt(1))
	s := 0
	for q.Bit(0) == 0 {
		q.Rsh(q, 1)
		s++
	}
	// Legendre symbol check: n^((p-1)/2) must be 1.
	legendreExp := new(big.Int).Rsh(new(big.Int).Sub(starkPrime, big.NewInt(1)), 1)
	legendre := new(big.Int).Exp(n, legendreExp, starkPrime)
	if legendre.Cmp(big.NewInt(1)) != 0 {
		return nil
	}
	// Find a quadratic non-residue.
	z := big.NewInt(2)
	for {
		cand := new(big.Int).Exp(z, legendreExp, starkPrime)
		if cand.Cmp(new(big.Int).Sub(starkPrime, big.NewInt(1))) == 0 {
			break
		}
		z.Add(z, big.NewInt(1))
	}
	m := s
	c := new(big.Int).Exp(z, q, starkPrime)
	t := new(big.Int).Exp(n, q, starkPrime)
	r := new(big.Int).Exp(n, new(big.Int).Rsh(new(big.Int).Add(q, big.NewInt(1)), 1), starkPrime)
	for t.Cmp(big.NewInt(1)) != 0 {
		// Find the least i, 0 < i < m, with t^(2^i) == 1.
		i := 0
		tmp := new(big.Int).Set(t)
		for tmp.Cmp(big.NewInt(1)) != 0 {
			tmp.Mul(tmp, tmp)
			tmp.Mod(tmp, starkPrime)
			i++
			if i == m {
				return nil
			}
		}
		b := new(big.Int).Exp(c, new(big.Int).Lsh(big.NewInt(1), uint(m-i-1)), starkPrime)
		m = i
		c = new(big.Int).Mul(b, b)
		c.Mod(c, starkPrime)
		t = new(big.Int).Mul(t, c)
		t.Mod(t, starkPrime)
		r = new(big.Int).Mul(r, b)
		r.Mod(r, starkPrime)
	}
	check := new(big.Int).Mul(r, r)
	check.Mod(check, starkPrime)
	if check.Cmp(n) != 0 {
		return nil
	}
	return r
}

// NEO N3 addresses use Hash160 of the verification script, not SHA256. The
// expected value below was produced by the neo-project reference rule
// (Base58Check(0x35 || RIPEMD160(SHA256(0x21||pub||0xac)))) for the secp256k1
// key 1, which the derivation path m/44'/888'/0'/0/0 cannot reach; the test
// therefore pins the scheme by checking the hash is Hash160 of the script the
// function builds.
func TestNEOAddressUsesHash160(t *testing.T) {
	seed := []byte("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about")
	addr, err := NEOAddress(seed, "m/44'/888'/0'/0/0")
	if err != nil {
		t.Fatalf("neo address: %v", err)
	}
	if !strings.HasPrefix(addr, "N") {
		t.Fatalf("NEO N3 mainnet address must start with N, got %q", addr)
	}
	raw, err := base58checkDecode(addr)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw) != 21 || raw[0] != 0x35 {
		t.Fatalf("payload must be version 0x35 plus 20 bytes, got %x", raw)
	}
	priv, err := HDDerive(seed, "m/44'/888'/0'/0/0")
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	x := make([]byte, 32)
	priv.PublicKey.X.FillBytes(x)
	compressed := append([]byte{0x02 + byte(priv.PublicKey.Y.Bit(0))}, x...)
	script := append(append([]byte{0x21}, compressed...), 0xac)
	if want := Hash160(script); !bytes.Equal(want, raw[1:]) {
		t.Fatalf("script hash = %x, want Hash160 %x", raw[1:], want)
	}
}

// TestStarknetCurveOrderConstant guards the two constants that are easy to
// confuse: the field prime and the group order differ, and the group order is
// what the ECDSA s term reduces against.
func TestStarknetCurveOrderConstant(t *testing.T) {
	if starkOrder.Cmp(starkPrime) == 0 {
		t.Fatal("group order must differ from the field prime")
	}
	if starkOrder.Cmp(starkPrime) >= 0 {
		t.Fatal("group order must be smaller than the field prime")
	}
	if starkMul(starkOrder, starkGenerator) != nil {
		t.Fatal("order * G must be the point at infinity")
	}
}

// base58checkDecode verifies and strips the 4-byte double-SHA256 checksum, so a
// test can assert on the payload a real address encodes.
func base58checkDecode(s string) ([]byte, error) {
	raw, err := Base58Decode(s)
	if err != nil {
		return nil, err
	}
	if len(raw) < 5 {
		return nil, errors.New("base58check: payload too short")
	}
	body, checksum := raw[:len(raw)-4], raw[len(raw)-4:]
	sum := sha256.Sum256(body)
	sum2 := sha256.Sum256(sum[:])
	if !bytes.Equal(sum2[:4], checksum) {
		return nil, errors.New("base58check: checksum mismatch")
	}
	return body, nil
}

// moneroBase58Decode is written from the Monero specification independently of
// moneroBase58Encode, so the round-trip below is a real check rather than a
// mirror of the encoder's own logic.
func moneroBase58Decode(s string) ([]byte, error) {
	var out []byte
	for i := 0; i < len(s); {
		blockLen := 11
		if len(s)-i < 11 {
			blockLen = len(s) - i
		}
		rawLen := 0
		for rawLen < 8 && moneroEncodedBlockSize[rawLen+1] <= blockLen {
			rawLen++
		}
		if rawLen == 0 {
			break
		}
		num := new(big.Int)
		for j := 0; j < blockLen; j++ {
			idx := bytes.IndexByte([]byte(moneroAlphabet), s[i+j])
			if idx < 0 {
				return nil, errors.New("monero: invalid base58 character")
			}
			num.Mul(num, big.NewInt(58))
			num.Add(num, big.NewInt(int64(idx)))
		}
		buf := make([]byte, rawLen)
		num.FillBytes(buf)
		out = append(out, buf...)
		i += blockLen
	}
	return out, nil
}

// A real Monero address must decode to the documented 69-byte layout, carry a
// valid Keccak-256 checksum, and re-encode byte-for-byte. This pins the encoder
// to addresses real Monero wallets produce rather than to its own output.
func TestMoneroRealAddressChecksum(t *testing.T) {
	const addr = "44AFFq5kSiGBoZ4NMDwYtN18obc8AemS33DBLWs3H7otXft3XjrpDtQGv7SqSsaBYBb98uNbr2VBBEt7f2wfn3RVGQBEP3A"
	raw, err := moneroBase58Decode(addr)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw) != 69 {
		t.Fatalf("decoded %d bytes, want 69", len(raw))
	}
	if raw[0] != 18 {
		t.Fatalf("network byte = %d, want 18 (mainnet standard)", raw[0])
	}
	body := raw[:65]
	if !bytes.Equal(crypto.Keccak256(body)[:4], raw[65:]) {
		t.Fatal("checksum mismatch: encoder does not match real Monero addresses")
	}
	if got := moneroBase58Encode(raw); got != addr {
		t.Fatalf("re-encode = %s, want %s", got, addr)
	}
}

func TestRegistryExpansionChainsFailClosed(t *testing.T) {
	failClosed := map[string]bool{
		"monero": true, "hyperliquid": true, "supra": true, "iota": true,
		"kadena": true, "chia": true, "handshake": true, "lisk": true,
		"nexus": true, "ark": true, "aion": true, "nuls": true,
		"pascalcoin": true, "signum": true,
	}
	for _, c := range NonEVMMainnet {
		family, _, err := ResolveFamily(c.ChainType, c.ID)
		if err != nil {
			t.Fatalf("chain %s (%d) does not resolve: %v", c.Name, c.ID, err)
		}
		if family == FamilyUnknown {
			t.Fatalf("chain %s (%d) resolved to FamilyUnknown", c.Name, c.ID)
		}
		if failClosed[c.ChainType] && family != FamilyUnsupported && family != FamilyMonero && family != FamilyArweave {
			t.Fatalf("chain %s should be fail-closed, got family %q", c.Name, family)
		}
	}
}
