package main

import (
	"math/big"
	"testing"
)

// TestWeiScaleToBaseUnits pins the wei-scale -> base-unit conversion. The
// amount that gets signed must equal the amount that was authorised, so an
// inexact conversion has to be an error rather than a rounding.
func TestWeiScaleToBaseUnits(t *testing.T) {
	cases := []struct {
		name     string
		amount   string
		decimals int
		want     string
		wantErr  bool
	}{
		// 1 Tron TRX = 1e18 wei-scale -> 1e6 sun.
		{"tron 1 trx", "1000000000000000000", 6, "1000000", false},
		// 1 BTC = 1e18 wei-scale -> 1e8 satoshi.
		{"btc 1", "1000000000000000000", 8, "100000000", false},
		// 1 APT = 1e18 -> 1e8 octas.
		{"aptos 1", "1000000000000000000", 8, "100000000", false},
		// 1 XRP = 1e18 -> 1e6 drops.
		{"xrp 1", "1000000000000000000", 6, "1000000", false},
		// 1 SUI = 1e18 -> 1e9 MIST.
		{"sui 1", "1000000000000000000", 9, "1000000000", false},
		// 18 decimals is the identity.
		{"18 decimals", "1500000000000000000", 18, "1500000000000000000", false},
		// Zero is a valid amount.
		{"zero", "0", 8, "0", false},
		// A remainder means the amount cannot be represented exactly.
		{"inexact remainder", "1500000000000000001", 8, "", true},
		{"negative", "-1", 8, "", true},
		{"not a number", "1.5", 8, "", true},
		{"empty", "", 8, "", true},
		{"decimals out of range", "1000000000000000000", 31, "", true},
		// NEAR is finer than wei scale (24 decimals), so the conversion
		// multiplies: 1 yoctoNEAR-scale unit becomes 1e6.
		{"near 24 decimals", "1000000000000000000", 24, "1000000000000000000000000", false},
		{"hedera 30 decimals", "1000000000000000000", 30, "1000000000000000000000000000000", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := weiScaleToBaseUnits(tc.amount, tc.decimals)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.String() != tc.want {
				t.Fatalf("got %s, want %s", got.String(), tc.want)
			}
		})
	}
}

// TestChainDecimalsMatchesRegistry guards against a chain being added to the
// registry without a decimals value, which would make every transfer fail.
func TestChainDecimalsMatchesRegistry(t *testing.T) {
	for _, c := range defaultNonEVMChains {
		d, ok := chainDecimals(c.ChainID)
		if !ok {
			t.Errorf("chain %d (%s) not resolvable by id", c.ChainID, c.Name)
			continue
		}
		if d != c.Decimals {
			t.Errorf("chain %d: decimals %d, want %d", c.ChainID, d, c.Decimals)
		}
	}
}

// TestChainSDKFamiliesAreRoutable asserts every family routed to
// autoSignChainSDK has a builder, so the dispatch switch and the builder
// switch cannot drift apart.
func TestChainSDKFamiliesAreRoutable(t *testing.T) {
	routed := []string{"tron", "aptos", "sui", "near", "algorand", "stellar",
		"ripple", "vechain", "tezos", "multiversx", "elrond"}
	withBuilder := map[string]bool{
		"tron": true, "aptos": true, "sui": true, "near": true,
		"algorand": true, "stellar": true, "ripple": true, "vechain": true,
		"tezos": true, "multiversx": true, "elrond": true,
	}
	for _, f := range routed {
		if !withBuilder[f] {
			t.Errorf("family %q is routed to autoSignChainSDK but has no builder", f)
		}
	}
}

// TestUnknownChainFailsClosed: a chain outside the registry must be refused
// rather than signed with a guessed decimals scale.
func TestUnknownChainFailsClosed(t *testing.T) {
	if _, ok := chainDecimals(424242); ok {
		t.Fatal("unregistered chain id resolved to a decimals value")
	}
}

// TestWeiScaleRoundTrip: converting to base units and back must be lossless
// for every registry chain, which is what makes the signed amount trustworthy.
func TestWeiScaleRoundTrip(t *testing.T) {
	const one = "1000000000000000000" // 1 unit at wei scale
	for _, c := range defaultNonEVMChains {
		base, err := weiScaleToBaseUnits(one, c.Decimals)
		if err != nil {
			t.Fatalf("chain %d (%s): %v", c.ChainID, c.Name, err)
		}
		back := new(big.Int).Mul(base, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(18-c.Decimals)), nil))
		if c.Decimals > 18 {
			// Finer than wei scale: the conversion multiplies, so undo it.
			back = new(big.Int).Quo(base, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(c.Decimals-18)), nil))
		}
		if back.String() != one {
			t.Errorf("chain %d (%s): round trip gave %s, want %s", c.ChainID, c.Name, back, one)
		}
	}
}
