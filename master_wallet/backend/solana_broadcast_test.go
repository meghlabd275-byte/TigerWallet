package main

import (
        "bytes"
        "crypto/ed25519"
        "testing"
)

func TestBase58DecodeStrict_RoundTrip(t *testing.T) {
        pub := make(ed25519.PublicKey, ed25519.PublicKeySize)
        for i := range pub {
                pub[i] = byte(i + 1)
        }
        enc := base58Encode(pub)
        dec, err := base58DecodeStrict(enc)
        if err != nil {
                t.Fatalf("base58DecodeStrict: %v", err)
        }
        if !bytes.Equal(dec, pub) {
                t.Fatalf("round-trip mismatch: %x != %x", dec, pub)
        }
}

func TestBase58DecodeStrict_Invalid(t *testing.T) {
        if _, err := base58DecodeStrict("0invalid"); err == nil {
                t.Fatalf("expected error for invalid base58 char")
        }
}

func TestBuildSolanaTransferMessage_WellFormed(t *testing.T) {
        from := make(ed25519.PublicKey, ed25519.PublicKeySize)
        to := make(ed25519.PublicKey, ed25519.PublicKeySize)
        for i := 0; i < ed25519.PublicKeySize; i++ {
                from[i] = byte(11)
                to[i] = byte(22)
        }
        blockhash := make([]byte, 32)
        for i := range blockhash {
                blockhash[i] = byte(33)
        }
        msg, err := buildSolanaTransferMessage(from, to, 1000000000, blockhash)
        if err != nil {
                t.Fatalf("buildSolanaTransferMessage: %v", err)
        }
        if len(msg) == 0 {
                t.Fatalf("empty message")
        }
        // header: 1 required sig, 0 readonly signed, 1 readonly unsigned
        if msg[0] != 1 || msg[1] != 0 || msg[2] != 1 {
                t.Fatalf("bad message header: %x", msg[:3])
        }
}

func TestBuildSolanaTransferMessage_BadBlockhash(t *testing.T) {
        from := make(ed25519.PublicKey, ed25519.PublicKeySize)
        to := make(ed25519.PublicKey, ed25519.PublicKeySize)
        if _, err := buildSolanaTransferMessage(from, to, 1, make([]byte, 31)); err == nil {
                t.Fatalf("expected error for 31-byte blockhash")
        }
}