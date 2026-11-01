package main

import (
	"math/big"
	"testing"
)

// A recorded eth_feeHistory response from an Ethereum mainnet node. Using a
// real response keeps the test honest about the wire format, including the
// trailing base fee for the next block.
func mainnetFeeHistory() feeHistory {
	return feeHistory{
		OldestBlock: "0x1919a81",
		BaseFeePerGas: []string{
			"0x27d9dd8", "0x2698985", "0x257a11c", "0x29c0b59", "0x2879596", "0x2935de1",
		},
		Reward: [][]string{
			{"0xa3", "0x989680", "0x5ad66500"},
			{"0x1", "0x86da81", "0x45d4c081"},
			{"0x186a0", "0xab8e4", "0x1c8068dc"},
			{"0xf4240", "0x2faf081", "0x77359400"},
			{"0xb0", "0xb8f0a0", "0x6b49d200"},
		},
	}
}

func TestComputeTiersUsesObservedPercentiles(t *testing.T) {
	low, market, fast, err := computeTiers(mainnetFeeHistory())
	if err != nil {
		t.Fatalf("computeTiers: %v", err)
	}

	// Tiers must be ordered: a higher percentile tip costs more.
	if low.MaxFeeWei.Cmp(market.MaxFeeWei) >= 0 {
		t.Errorf("slow max fee %v should be below standard %v", low.MaxFeeWei, market.MaxFeeWei)
	}
	if market.MaxFeeWei.Cmp(fast.MaxFeeWei) >= 0 {
		t.Errorf("standard max fee %v should be below fast %v", market.MaxFeeWei, fast.MaxFeeWei)
	}
	if low.PriorityFeeWei.Cmp(market.PriorityFeeWei) > 0 {
		t.Errorf("slow tip %v should not exceed standard tip %v", low.PriorityFeeWei, market.PriorityFeeWei)
	}
	if market.PriorityFeeWei.Cmp(fast.PriorityFeeWei) > 0 {
		t.Errorf("standard tip %v should not exceed fast tip %v", market.PriorityFeeWei, fast.PriorityFeeWei)
	}
}

// The base fee used must be the LAST entry: that is the next block's base fee.
// Using an earlier entry would price the transaction against a stale base.
func TestComputeTiersUsesNextBlockBaseFee(t *testing.T) {
	history := mainnetFeeHistory()
	low, _, _, err := computeTiers(history)
	if err != nil {
		t.Fatalf("computeTiers: %v", err)
	}

	want, _ := new(big.Int).SetString("2935de1", 16)
	if low.BaseFeeWei.Cmp(want) != 0 {
		t.Errorf("base fee = %v, want %v (the trailing entry)", low.BaseFeeWei, want)
	}
}

// The tip is the mean of the percentile column, so a single outlier block
// cannot dominate the recommendation.
func TestComputeTiersAveragesThePercentileColumn(t *testing.T) {
	_, market, _, err := computeTiers(mainnetFeeHistory())
	if err != nil {
		t.Fatalf("computeTiers: %v", err)
	}

	// Median column: 0x989680, 0x86da81, 0xab8e4, 0x2faf081, 0xb8f0a0.
	tips := []int64{0x989680, 0x86da81, 0xab8e4, 0x2faf081, 0xb8f0a0}
	sum := big.NewInt(0)
	for _, tip := range tips {
		sum.Add(sum, big.NewInt(tip))
	}
	want := new(big.Int).Div(sum, big.NewInt(int64(len(tips))))

	if market.PriorityFeeWei.Cmp(want) != 0 {
		t.Errorf("standard tip = %v, want %v", market.PriorityFeeWei, want)
	}
}

// Max fee must always cover the base fee plus the tip, or the transaction is
// rejected at submission for being unable to pay the base fee.
func TestComputeTiersMaxFeeCoversBaseFeeAndTip(t *testing.T) {
	low, market, fast, err := computeTiers(mainnetFeeHistory())
	if err != nil {
		t.Fatalf("computeTiers: %v", err)
	}

	for name, tier := range map[string]GasTier{"slow": low, "standard": market, "fast": fast} {
		required := new(big.Int).Add(tier.BaseFeeWei, tier.PriorityFeeWei)
		if tier.MaxFeeWei.Cmp(required) < 0 {
			t.Errorf("%s max fee %v does not cover base %v + tip %v",
				name, tier.MaxFeeWei, tier.BaseFeeWei, tier.PriorityFeeWei)
		}
	}
}

func TestComputeTiersRejectsEmptyHistory(t *testing.T) {
	if _, _, _, err := computeTiers(feeHistory{}); err == nil {
		t.Fatal("empty fee history must be an error, not a zero-price tier")
	}
}

// A node that returns fewer percentile columns than requested must fail rather
// than silently fall back to a fabricated tip.
func TestComputeTiersRejectsMissingPercentileColumns(t *testing.T) {
	history := mainnetFeeHistory()
	for i := range history.Reward {
		history.Reward[i] = history.Reward[i][:1] // only the 10th percentile
	}
	if _, _, _, err := computeTiers(history); err == nil {
		t.Fatal("missing percentile columns must be an error")
	}
}

// Malformed hex must not be read as zero.
func TestComputeTiersRejectsMalformedBaseFee(t *testing.T) {
	history := mainnetFeeHistory()
	history.BaseFeePerGas = []string{"not-hex"}
	if _, _, _, err := computeTiers(history); err == nil {
		t.Fatal("unparsable base fee must be an error")
	}
}

func TestHexToBig(t *testing.T) {
	cases := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"0x10", 16, true},
		{"0X10", 16, true},
		{"10", 10, true},
		{"0x0", 0, true},
		{"", 0, false},
		{"zz", 0, false},
	}
	for _, tc := range cases {
		got, ok := hexToBig(tc.in)
		if ok != tc.ok {
			t.Errorf("hexToBig(%q) ok = %v, want %v", tc.in, ok, tc.ok)
			continue
		}
		if ok && got.Int64() != tc.want {
			t.Errorf("hexToBig(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
