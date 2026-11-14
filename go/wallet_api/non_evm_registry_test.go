package main

import (
	"strings"
	"testing"

	"github.com/tigerwallet/chainsdk"
)

// non_evm_registry_test.go — cross-registry invariants. This lives in
// wallet_api because the EVM registry is defined here; chainsdk owns the
// non-EVM registry.

// audit: IOTA EVM and HyperEVM were listed in both the EVM and the non-EVM
// registries, so the same network appeared twice under different chain ids.
//
// A chain name may legitimately appear in both registries when it really is two
// networks — Injective is a Cosmos chain and inEVM is a separate EVM chain — so
// the allowed pairs are listed explicitly.
func TestNonEvmRegistryHasNoEVMDuplicates(t *testing.T) {
	legitimate := map[string]bool{
		"injective": true, // Cosmos chain vs inEVM (EVM chain)
	}
	evmNames := map[string]bool{}
	for _, c := range evmMainnet {
		evmNames[strings.ToLower(c.Name)] = true
	}
	for _, c := range chainsdk.NonEVMMainnet {
		name := strings.ToLower(c.Name)
		if evmNames[name] && !legitimate[name] {
			t.Errorf("non-EVM registry entry %q (id %d) duplicates an EVM registry entry",
				c.Name, c.ID)
		}
	}
}
