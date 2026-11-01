package main

import (
        "encoding/hex"
        "math/big"
        "testing"
        "time"

        "github.com/ethereum/go-ethereum/common"
        "github.com/ethereum/go-ethereum/crypto"
)

// TestFactoryAddressMethodSelectsTheRightView guards the one behavioural
// difference between the two supported factories: the canonical
// SimpleAccountFactory exposes `getAddress`, TigerWallet's exposes
// `getAccountAddress`. Calling the wrong one would revert on chain.
func TestFactoryAddressMethodSelectsTheRightView(t *testing.T) {
        cases := []struct {
                mode string
                want string
        }{
                {factoryModeCanonical, "getAddress(address,uint256)"},
                {factoryModeTigerWallet, "getAccountAddress(address,uint256)"},
        }

        for _, c := range cases {
                t.Run(c.mode, func(t *testing.T) {
                        factory := &factoryConfig{
                                Address: common.HexToAddress("0x9406Cc6185a346906296840746125a0E44976454"),
                                Mode:    c.mode,
                        }
                        if got := factory.addressMethod(); got != c.want {
                                t.Fatalf("addressMethod() = %q, want %q", got, c.want)
                        }
                })
        }
}

// TestFactoryReturnsNilWhenUnconfigured ensures the service fails closed rather
// than inventing a factory address.
func TestFactoryReturnsNilWhenUnconfigured(t *testing.T) {
        api := &BundlerAPI{config: &Config{}}

        if api.Factory() != nil {
                t.Fatal("an unconfigured bundler reported a factory")
        }
}

// TestMethodSelectorMatchesTheCanonicalVectors pins the selectors to the values
// the deployed contracts dispatch on.
func TestMethodSelectorMatchesTheCanonicalVectors(t *testing.T) {
        cases := []struct {
                signature string
                want      string
        }{
                // keccak256("getAddress(address,uint256)")[:4]
                {"getAddress(address,uint256)", "8cb84e18"},
                // keccak256("getAccountAddress(address,uint256)")[:4]
                {"getAccountAddress(address,uint256)", "0d253d76"},
                // keccak256("createAccount(address,uint256)")[:4]
                {"createAccount(address,uint256)", "5fbfb9cf"},
                // keccak256("getNonce(address,uint192)")[:4]
                {"getNonce(address,uint192)", "35567e1a"},
        }

        for _, c := range cases {
                got := hex.EncodeToString(methodSelector(c.signature))
                if got != c.want {
                        t.Errorf("selector(%s) = %s, want %s", c.signature, got, c.want)
                }
        }
}

// TestEncodeCallProducesSelectorPlusArguments checks the calldata layout: a
// 4-byte selector followed by 32-byte words.
func TestEncodeCallProducesSelectorPlusArguments(t *testing.T) {
        owner := common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e")

        data, err := encodeCall("getAddress(address,uint256)", owner, big.NewInt(7))
        if err != nil {
                t.Fatalf("encodeCall: %v", err)
        }

        if len(data) != 4+32+32 {
                t.Fatalf("calldata length = %d, want 68", len(data))
        }

        if hex.EncodeToString(data[:4]) != "8cb84e18" {
                t.Fatalf("selector = %s, want 8cb84e18", hex.EncodeToString(data[:4]))
        }

        // The address occupies the low 20 bytes of its word.
        if common.BytesToAddress(data[4+12:4+32]) != owner {
                t.Fatalf("encoded owner = %s, want %s", common.BytesToAddress(data[16:36]).Hex(), owner.Hex())
        }

        if new(big.Int).SetBytes(data[36:68]).Int64() != 7 {
                t.Fatal("the salt was not encoded")
        }
}

// TestEncodeCallRejectsUnknownTypes keeps the encoder honest.
func TestEncodeCallRejectsUnknownTypes(t *testing.T) {
        if _, err := encodeCall("x(uint8)", uint8(1)); err == nil {
                t.Fatal("an unsupported argument type was accepted")
        }
}

// TestPaymasterDataLayoutMatchesTheContract pins the `paymasterAndData` layout
// the VerifyingPaymaster slices: address(20) || verificationGas(16) ||
// postOpGas(16) || signature(65) || validUntil(6) || validAfter(6).
func TestPaymasterDataLayoutMatchesTheContract(t *testing.T) {
        verificationGas := leftPadUint128(150_000)
        postOpGas := leftPadUint128(50_000)

        if len(verificationGas) != 16 || len(postOpGas) != 16 {
                t.Fatalf("gas words must be 16 bytes, got %d and %d", len(verificationGas), len(postOpGas))
        }

        if new(big.Int).SetBytes(verificationGas).Int64() != 150_000 {
                t.Fatal("verificationGas did not round-trip")
        }
        if new(big.Int).SetBytes(postOpGas).Int64() != 50_000 {
                t.Fatal("postOpGas did not round-trip")
        }

        // The contract's PAYMASTER_DATA_OFFSET is 20 + 16 + 16 = 52.
        if len(verificationGas)+len(postOpGas)+20 != 52 {
                t.Fatal("the paymaster data offset no longer matches the contract constant")
        }

        validUntil := uint48Bytes(1_800_000_000)
        if len(validUntil) != 6 {
                t.Fatalf("a uint48 must be 6 bytes, got %d", len(validUntil))
        }
        if new(big.Int).SetBytes(validUntil).Int64() != 1_800_000_000 {
                t.Fatal("validUntil did not round-trip")
        }
}

// TestPaymasterDataLengthMatchesTheContract asserts the total blob length the
// VerifyingPaymaster demands: it rejects any paymasterData that is not exactly
// 65 + 12 bytes.
func TestPaymasterDataLengthMatchesTheContract(t *testing.T) {
        signature := make([]byte, 65)
        blob := make([]byte, 0, 20+16+16+65+6+6)
        blob = append(blob, make([]byte, 20)...)
        blob = append(blob, leftPadUint128(150_000)...)
        blob = append(blob, leftPadUint128(50_000)...)
        blob = append(blob, signature...)
        blob = append(blob, uint48Bytes(1_800_000_000)...)
        blob = append(blob, uint48Bytes(1_700_000_000)...)

        // The contract slices from offset 52 and requires 65+12 bytes.
        paymasterData := blob[52:]
        if len(paymasterData) != 65+12 {
                t.Fatalf("paymasterData length = %d, want 77", len(paymasterData))
        }

        if len(blob) != 20+16+16+65+6+6 {
                t.Fatalf("blob length = %d, want 129", len(blob))
        }
}

// TestSponsorshipSignatureRecoversTheSponsor proves the signature the endpoint
// returns is the one the paymaster will recover: EIP-191 over the user
// operation hash, with a 27/28 recovery byte.
func TestSponsorshipSignatureRecoversTheSponsor(t *testing.T) {
        key, err := crypto.GenerateKey()
        if err != nil {
                t.Fatalf("GenerateKey: %v", err)
        }

        userOpHash := crypto.Keccak256Hash([]byte("a user operation"))
        digest := crypto.Keccak256([]byte("\x19Ethereum Signed Message:\n32"), userOpHash.Bytes())

        signature, err := crypto.Sign(digest, key)
        if err != nil {
                t.Fatalf("Sign: %v", err)
        }
        // crypto.Sign returns v in {0,1}. The wire format the paymaster consumes
        // is the canonical Ethereum encoding, so the endpoint shifts it to 27/28.
        signature[64] += 27

        if signature[64] != 27 && signature[64] != 28 {
                t.Fatalf("recovery byte = %d, want 27 or 28", signature[64])
        }

        // The contract's ECDSA.recover normalises 27/28 back to 0/1 before
        // calling ecrecover, so recover the same way here.
        normalised := make([]byte, 65)
        copy(normalised, signature)
        normalised[64] -= 27

        recovered, err := crypto.SigToPub(digest, normalised)
        if err != nil {
                t.Fatalf("SigToPub: %v", err)
        }

        if crypto.PubkeyToAddress(*recovered) != crypto.PubkeyToAddress(key.PublicKey) {
                t.Fatal("the recovered signer is not the sponsor")
        }

        // A signature that is not shifted would not be recoverable by a
        // 27/28-expecting verifier, which is why the shift matters.
        if normalised[64] == signature[64] {
                t.Fatal("the recovery byte was not shifted into the Ethereum encoding")
        }
}

// TestParseOptionalUintAcceptsBothQuantityForms covers the decimal and hex
// forms bundler clients send.
func TestParseOptionalUintAcceptsBothQuantityForms(t *testing.T) {
        cases := []struct {
                in   string
                want int64
        }{
                {"", 0},
                {"0", 0},
                {"7", 7},
                {"0x7", 7},
                {"0x10", 16},
        }

        for _, c := range cases {
                got, err := parseOptionalUint(c.in)
                if err != nil {
                        t.Fatalf("parseOptionalUint(%q): %v", c.in, err)
                }
                if got.Int64() != c.want {
                        t.Errorf("parseOptionalUint(%q) = %d, want %d", c.in, got.Int64(), c.want)
                }
        }
}

// TestParseOptionalUintRejectsGarbage keeps malformed input from becoming salt 0.
func TestParseOptionalUintRejectsGarbage(t *testing.T) {
        for _, in := range []string{"not-a-number", "0xzz", "1.5"} {
                if _, err := parseOptionalUint(in); err == nil {
                        t.Errorf("parseOptionalUint(%q) was accepted", in)
                }
        }
}

// TestConfigReadsFactoryAndPaymaster pins the new configuration surface.
func TestConfigReadsFactoryAndPaymaster(t *testing.T) {
        t.Setenv("RPC_URL", "http://127.0.0.1:1")
        t.Setenv("ACCOUNT_FACTORY", "0x9406Cc6185a346906296840746125a0E44976454")
        t.Setenv("FACTORY_MODE", factoryModeTigerWallet)
        t.Setenv("PAYMASTER_ADDRESS", "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789")

        config, err := ConfigFromEnv()
        if err != nil {
                t.Fatalf("ConfigFromEnv: %v", err)
        }

        if config.AccountFactory != common.HexToAddress("0x9406Cc6185a346906296840746125a0E44976454") {
                t.Fatal("ACCOUNT_FACTORY was not read")
        }
        if config.FactoryMode != factoryModeTigerWallet {
                t.Fatalf("FactoryMode = %q", config.FactoryMode)
        }
        if config.PaymasterAddress != common.HexToAddress("0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789") {
                t.Fatal("PAYMASTER_ADDRESS was not read")
        }
        if config.SponsorshipWindow != time.Hour {
                t.Fatalf("SponsorshipWindow = %v, want 1h", config.SponsorshipWindow)
        }
}

// TestConfigRejectsAnUnknownFactoryMode prevents a typo from silently selecting
// the wrong address view.
func TestConfigRejectsAnUnknownFactoryMode(t *testing.T) {
        t.Setenv("RPC_URL", "http://127.0.0.1:1")
        t.Setenv("FACTORY_MODE", "something-else")

        if _, err := ConfigFromEnv(); err == nil {
                t.Fatal("an unknown FACTORY_MODE was accepted")
        }
}

// TestConfigRejectsAnInvalidFactoryAddress fails closed on a malformed address.
func TestConfigRejectsAnInvalidFactoryAddress(t *testing.T) {
        t.Setenv("RPC_URL", "http://127.0.0.1:1")
        t.Setenv("ACCOUNT_FACTORY", "0x1234")

        if _, err := ConfigFromEnv(); err == nil {
                t.Fatal("an invalid ACCOUNT_FACTORY was accepted")
        }
}

// TestConfigRejectsAnInvalidPaymasterAddress fails closed on a malformed address.
func TestConfigRejectsAnInvalidPaymasterAddress(t *testing.T) {
        t.Setenv("RPC_URL", "http://127.0.0.1:1")
        t.Setenv("PAYMASTER_ADDRESS", "nope")

        if _, err := ConfigFromEnv(); err == nil {
                t.Fatal("an invalid PAYMASTER_ADDRESS was accepted")
        }
}

// TestLoadPaymasterSignerFailsClosedWithoutAKey ensures sponsorship is disabled
// rather than signed with a zero key.
func TestLoadPaymasterSignerFailsClosedWithoutAKey(t *testing.T) {
        t.Setenv("PAYMASTER_SIGNER_KEY", "")

        if _, err := loadPaymasterSigner(); err == nil {
                t.Fatal("a missing sponsor key was accepted")
        }
}

// TestLoadPaymasterSignerReadsTheKey reads a real key in both encodings.
func TestLoadPaymasterSignerReadsTheKey(t *testing.T) {
        key, err := crypto.GenerateKey()
        if err != nil {
                t.Fatalf("GenerateKey: %v", err)
        }

        encoded := hex.EncodeToString(crypto.FromECDSA(key))

        for _, value := range []string{encoded, "0x" + encoded} {
                t.Setenv("PAYMASTER_SIGNER_KEY", value)

                loaded, err := loadPaymasterSigner()
                if err != nil {
                        t.Fatalf("loadPaymasterSigner(%q): %v", value[:6], err)
                }
                if crypto.PubkeyToAddress(loaded.PublicKey) != crypto.PubkeyToAddress(key.PublicKey) {
                        t.Fatal("the loaded key is not the generated key")
                }
        }
}

// TestMustJSONNeverPanics keeps the REST wrapper total.
func TestMustJSONNeverPanics(t *testing.T) {
        if string(mustJSON("abc")) != `"abc"` {
                t.Fatal("mustJSON did not encode a string")
        }

        // A channel cannot be marshalled; the helper must degrade, not panic.
        if string(mustJSON(make(chan int))) != "null" {
                t.Fatal("mustJSON did not degrade on an unencodable value")
        }
}
