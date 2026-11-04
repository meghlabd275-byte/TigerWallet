package main

// portfolio_calc_test.go — Exercises the analytics the service actually
// returns. These values used to be constants (a fixed 40/25/20/10/5 allocation
// and a risk score of 5.5), so the tests assert that the numbers now depend on
// the portfolio they are computed from.

import (
	"math"
	"testing"
)

func newTestService() *PortfolioService {
	return NewPortfolioService()
}

// oneAssetPortfolio is a single ETH position worth 1000 USD.
func oneAssetPortfolio() *Portfolio {
	p := &Portfolio{
		UserID:        "u1",
		TotalValueUSD: 1000,
		Positions: []Position{
			{ID: "w1", Type: "wallet", Chain: "ethereum", Symbol: "ETH",
				Balance: 1, ValueUSD: 1000, Priced: true},
		},
	}
	p.Assets = []AssetSummary{{Symbol: "ETH", Amount: 1, ValueUSD: 1000, Percent: 100}}
	p.Chains = []ChainSummary{{Chain: "ethereum", ValueUSD: 1000, Percent: 100}}
	return p
}

// TestAllocationSumsToTotal is the property the fixed table broke: a hardcoded
// split always summed to the total by construction, but described assets the
// user did not hold.
func TestAllocationSumsToTotal(t *testing.T) {
	s := newTestService()
	p := &Portfolio{
		UserID:        "u2",
		TotalValueUSD: 300,
		Positions: []Position{
			{Type: "wallet", Chain: "ethereum", Symbol: "ETH", ValueUSD: 100, Priced: true},
			{Type: "lp", Chain: "polygon", Symbol: "USDC", ValueUSD: 150, Priced: true},
			{Type: "staking", Chain: "cosmos", Symbol: "ATOM", ValueUSD: 50, Priced: true},
		},
	}
	sum := 0.0
	for _, a := range s.allocationByType(p) {
		sum += a.Value
	}
	if math.Abs(sum-300) > 0.001 {
		t.Errorf("allocation sums to %v, want 300", sum)
	}
}

// TestAllocationUsesPositionTypes verifies the categories come from the
// holdings rather than a hardcoded DeFi/Staking/Trading/NFT/Other table.
func TestAllocationUsesPositionTypes(t *testing.T) {
	s := newTestService()
	p := &Portfolio{
		UserID:        "u3",
		TotalValueUSD: 200,
		Positions: []Position{
			{Type: "wallet", Chain: "ethereum", ValueUSD: 150, Priced: true},
			{Type: "lending", Chain: "ethereum", ValueUSD: 50, Priced: true},
		},
	}
	got := map[string]float64{}
	for _, a := range s.allocationByType(p) {
		got[a.Category] = a.Value
	}
	if len(got) != 2 {
		t.Fatalf("got %d categories %v, want 2 (wallet, lending)", len(got), got)
	}
	if got["wallet"] != 150 || got["lending"] != 50 {
		t.Errorf("allocation = %v, want wallet 150 and lending 50", got)
	}
	// The old table's categories must not appear for holdings that do not use them.
	for _, bogus := range []string{"DeFi", "Trading", "NFT", "Other"} {
		if _, ok := got[bogus]; ok {
			t.Errorf("allocation contains fabricated category %q", bogus)
		}
	}
}

// TestAllocationSortedByValue keeps the response stable across calls; the
// source is a map, whose iteration order is randomised.
func TestAllocationSortedByValue(t *testing.T) {
	s := newTestService()
	p := &Portfolio{
		UserID:        "u4",
		TotalValueUSD: 300,
		Positions: []Position{
			{Type: "wallet", ValueUSD: 50, Priced: true},
			{Type: "lp", ValueUSD: 200, Priced: true},
			{Type: "staking", ValueUSD: 50, Priced: true},
		},
	}
	all := s.allocationByType(p)
	for i := 1; i < len(all); i++ {
		if all[i-1].Value < all[i].Value {
			t.Errorf("allocation not sorted: %v", all)
			break
		}
	}
}

// TestRiskScoreDependsOnHoldings replaces the constant 5.5. A concentrated
// portfolio must score higher than a spread one, and neither may equal the old
// constant by construction.
func TestRiskScoreDependsOnHoldings(t *testing.T) {
	s := newTestService()

	concentrated := oneAssetPortfolio()
	concentratedScore := s.riskScore(concentrated)

	spread := &Portfolio{
		UserID:        "u5",
		TotalValueUSD: 1000,
		Positions: []Position{
			{Type: "wallet", Chain: "ethereum", Symbol: "ETH", ValueUSD: 250, Priced: true},
			{Type: "wallet", Chain: "polygon", Symbol: "USDC", ValueUSD: 250, Priced: true},
			{Type: "wallet", Chain: "solana", Symbol: "SOL", ValueUSD: 250, Priced: true},
			{Type: "wallet", Chain: "cosmos", Symbol: "ATOM", ValueUSD: 250, Priced: true},
		},
	}
	spread.Assets = []AssetSummary{
		{Symbol: "ETH", ValueUSD: 250}, {Symbol: "USDC", ValueUSD: 250},
		{Symbol: "SOL", ValueUSD: 250}, {Symbol: "ATOM", ValueUSD: 250},
	}
	spread.Chains = []ChainSummary{
		{Chain: "ethereum", ValueUSD: 250}, {Chain: "polygon", ValueUSD: 250},
		{Chain: "solana", ValueUSD: 250}, {Chain: "cosmos", ValueUSD: 250},
	}
	spreadScore := s.riskScore(spread)

	if concentratedScore <= spreadScore {
		t.Errorf("concentrated score %v should exceed spread score %v",
			concentratedScore, spreadScore)
	}
	if concentratedScore < 0 || concentratedScore > 10 {
		t.Errorf("risk score %v outside 0..10", concentratedScore)
	}
	if concentratedScore == 5.5 {
		t.Error("risk score equals the removed constant 5.5")
	}
}

// TestRiskScoreEmptyPortfolioIsZero: with nothing held there is no risk to
// report, and a non-zero score would be invented.
func TestRiskScoreEmptyPortfolioIsZero(t *testing.T) {
	s := newTestService()
	if got := s.riskScore(&Portfolio{UserID: "empty"}); got != 0 {
		t.Errorf("risk score for an empty portfolio = %v, want 0", got)
	}
}

// TestRiskScorePenalisesUnpricedHoldings: a position with no USD price hides
// value, which is a data-quality risk the score must reflect.
func TestRiskScorePenalisesUnpricedHoldings(t *testing.T) {
	s := newTestService()

	priced := oneAssetPortfolio()
	pricedScore := s.riskScore(priced)

	unpriced := oneAssetPortfolio()
	unpriced.Positions[0].Priced = false
	unpricedScore := s.riskScore(unpriced)

	if unpricedScore <= pricedScore {
		t.Errorf("unpriced score %v should exceed priced score %v", unpricedScore, pricedScore)
	}
}

// TestDiversificationRewardsSpreadAcrossChains.
func TestDiversificationRewardsSpreadAcrossChains(t *testing.T) {
	s := newTestService()

	one := oneAssetPortfolio()
	oneScore := s.diversificationScore(one)

	many := &Portfolio{
		UserID:        "u6",
		TotalValueUSD: 1000,
		Chains: []ChainSummary{
			{Chain: "a", ValueUSD: 100}, {Chain: "b", ValueUSD: 100},
			{Chain: "c", ValueUSD: 100}, {Chain: "d", ValueUSD: 100},
			{Chain: "e", ValueUSD: 100}, {Chain: "f", ValueUSD: 100},
			{Chain: "g", ValueUSD: 100}, {Chain: "h", ValueUSD: 100},
			{Chain: "i", ValueUSD: 100}, {Chain: "j", ValueUSD: 100},
		},
	}
	manyScore := s.diversificationScore(many)

	if manyScore <= oneScore {
		t.Errorf("ten-chain score %v should exceed one-chain score %v", manyScore, oneScore)
	}
	if manyScore > 100 {
		t.Errorf("diversification %v exceeds 100", manyScore)
	}
}

// TestDiversificationWeightsByValue: ten chains where one holds 99% of the
// value is not diversified, so the score must stay low.
func TestDiversificationWeightsByValue(t *testing.T) {
	s := newTestService()
	p := &Portfolio{
		UserID:        "u7",
		TotalValueUSD: 1000,
		Chains: []ChainSummary{
			{Chain: "a", ValueUSD: 991},
			{Chain: "b", ValueUSD: 1}, {Chain: "c", ValueUSD: 1},
			{Chain: "d", ValueUSD: 1}, {Chain: "e", ValueUSD: 1},
			{Chain: "f", ValueUSD: 1}, {Chain: "g", ValueUSD: 1},
			{Chain: "h", ValueUSD: 1}, {Chain: "i", ValueUSD: 1},
			{Chain: "j", ValueUSD: 1},
		},
	}
	if got := s.diversificationScore(p); got > 15 {
		t.Errorf("diversification for a 99%%-concentrated portfolio = %v, want <= 15", got)
	}
}

// TestPriceOfRejectsMissingAndNonPositivePrices. A price of 0 or a missing
// symbol must report "not priced" rather than being used as a valuation.
func TestPriceOfRejectsMissingAndNonPositivePrices(t *testing.T) {
	s := newTestService()

	if _, ok := s.priceOf("ETH"); ok {
		t.Error("priceOf on an empty cache reported a price")
	}
	if _, ok := s.priceOf(""); ok {
		t.Error("priceOf(\"\") reported a price")
	}

	s.prices["ETH"] = 2500
	if price, ok := s.priceOf("eth"); !ok || price != 2500 {
		t.Errorf("priceOf(eth) = %v,%v want 2500,true (lookup must be case-insensitive)",
			price, ok)
	}

	// A zero price is not a valuation. The previous code substituted 1.0 here,
	// which valued every unpriced balance at one dollar.
	s.prices["ZERO"] = 0
	if _, ok := s.priceOf("ZERO"); ok {
		t.Error("priceOf reported a zero price as valid")
	}
}

// TestPricesStaleBeforeAnyFetch: with no fetch yet, the cache must be reported
// stale so a client does not read a zero total as a real valuation.
func TestPricesStaleBeforeAnyFetch(t *testing.T) {
	s := newTestService()
	if !s.pricesStale() {
		t.Error("pricesStale() = false before any price fetch")
	}
}

// TestAssetSummaryIsKeyedBySymbol. This was the bug that made the analytics
// nonsense: updateAssetSummary keyed by Chain, so it produced a second copy of
// the chain summary and best/worst performer named a chain as an asset.
func TestAssetSummaryIsKeyedBySymbol(t *testing.T) {
	s := newTestService()
	p := &Portfolio{
		UserID:        "u8",
		TotalValueUSD: 300,
		Positions: []Position{
			// Two assets on the same chain.
			{Type: "wallet", Chain: "ethereum", Symbol: "ETH", Balance: 0.1,
				ValueUSD: 250, Priced: true},
			{Type: "wallet", Chain: "ethereum", Symbol: "USDC", Balance: 50,
				ValueUSD: 50, Priced: true},
		},
	}
	s.updateAssetSummary(p)

	if len(p.Assets) != 2 {
		t.Fatalf("got %d assets %v, want 2 (ETH, USDC)", len(p.Assets), p.Assets)
	}
	bySymbol := map[string]AssetSummary{}
	for _, a := range p.Assets {
		bySymbol[a.Symbol] = a
	}
	if a, ok := bySymbol["ETH"]; !ok || a.ValueUSD != 250 || a.Amount != 0.1 {
		t.Errorf("ETH summary = %+v, want value 250 amount 0.1", a)
	}
	if a, ok := bySymbol["USDC"]; !ok || a.ValueUSD != 50 {
		t.Errorf("USDC summary = %+v, want value 50", a)
	}
	if _, ok := bySymbol["ethereum"]; ok {
		t.Error("assets are keyed by chain; they must be keyed by symbol")
	}
}
