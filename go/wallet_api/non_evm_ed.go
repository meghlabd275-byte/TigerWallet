package main

// non_evm_ed.go — Ed25519 helpers shared by the Ed25519 account-chain
// families (solana, aptos, sui, near, stellar/pi, algorand, nano,
// multiversx, tezos, cardano, waves, ton, hedera, flow).

import (
	"errors"

	ed "golang.org/x/crypto/ed25519"
)

// edKeypair derives the SLIP-0010 Ed25519 keypair for the chain's BIP-44
// path. The private key is used internally only.
func edKeypair(seed []byte, path string) (ed.PrivateKey, ed.PublicKey, error) {
	s, err := slip10DeriveEd25519(seed, path)
	if err != nil {
		return nil, nil, err
	}
	priv := ed.NewKeyFromSeed(s)
	return priv, priv.Public().(ed.PublicKey), nil
}

// edSignMessage signs an arbitrary message with the chain's ed25519 key.
func edSignMessage(seed []byte, path string, msg []byte) (sig, pub []byte, err error) {
	priv, pub, err := edKeypair(seed, path)
	if err != nil {
		return nil, nil, err
	}
	return ed.Sign(priv, msg), pub, nil
}

// edPubKey helper — derive only the public key.
func edPubKey(seed []byte, path string) (ed.PublicKey, error) {
	_, pub, err := edKeypair(seed, path)
	if err != nil {
		return nil, err
	}
	return pub, nil
}

// pubFromEdScalarFromSeed derives an ed25519 pubkey from arbitrary 32-byte
// entropy (used by the curve25519/waves and cardano extended scalars).
func pubFromEdSeed(entropy []byte) (ed.PublicKey, error) {
	if len(entropy) != 32 {
		return nil, errors.New("ed25519 entropy must be 32 bytes")
	}
	priv := ed.NewKeyFromSeed(entropy)
	return priv.Public().(ed.PublicKey), nil
}
