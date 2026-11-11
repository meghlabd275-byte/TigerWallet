package main

import (
	"testing"
	"time"
)

func TestRiskScoreLowForTypicalRepeatPurchase(t *testing.T) {
	limits := &CardLimits{PerTransactionLimit: 500000}
	req := &AuthorizationRequest{
		Amount:           5000,
		Country:          "US",
		MerchantCategory: "5411",
		TerminalID:       "POS-1",
	}
	signals := riskSignals{
		RecentTxCount:       20,
		MeanAmount:          4800,
		MaxAmount:           9000,
		DistinctCountries:   1,
		FirstUse:            false,
		NewMerchantCategory: false,
		RoundAmount:         false,
		CardAgeHours:        24 * 365,
	}

	score := calculateRiskScore(req, limits, signals)
	if score >= riskReviewThreshold {
		t.Fatalf("a normal repeat purchase scored %d, expected below the review threshold %d", score, riskReviewThreshold)
	}
}

func TestRiskScoreHighForOutlierAmount(t *testing.T) {
	limits := &CardLimits{PerTransactionLimit: 500000}
	req := &AuthorizationRequest{
		Amount:           250000,
		Country:          "US",
		MerchantCategory: "5411",
		TerminalID:       "POS-1",
	}
	signals := riskSignals{
		RecentTxCount:     20,
		MeanAmount:        4800,
		MaxAmount:         9000,
		DistinctCountries: 1,
		CardAgeHours:      24 * 365,
	}

	score := calculateRiskScore(req, limits, signals)
	if score < riskReviewThreshold {
		t.Fatalf("a 50x outlier scored %d, expected at least %d", score, riskReviewThreshold)
	}
}

func TestRiskScoreDeclinesBrandNewCardWithHugeAmount(t *testing.T) {
	limits := &CardLimits{PerTransactionLimit: 100000}
	req := &AuthorizationRequest{
		Amount:           100000,
		Country:          "RU",
		MerchantCategory: "7995",
		TerminalID:       "",
	}
	signals := riskSignals{
		FirstUse:            true,
		NewMerchantCategory: true,
		RoundAmount:         true,
		CardAgeHours:        1,
	}

	score := calculateRiskScore(req, limits, signals)
	if score < riskDeclineThreshold {
		t.Fatalf("a brand-new card at its full limit scored %d, expected at least the decline threshold %d", score, riskDeclineThreshold)
	}
}

func TestRiskScoreIsBounded(t *testing.T) {
	limits := &CardLimits{PerTransactionLimit: 1}
	req := &AuthorizationRequest{Amount: 1000000, TerminalID: ""}
	signals := riskSignals{
		RecentTxCount:       5,
		MeanAmount:          1,
		FirstUse:            true,
		NewMerchantCategory: true,
		RoundAmount:         true,
		CardAgeHours:        0,
	}

	score := calculateRiskScore(req, limits, signals)
	if score > 100 {
		t.Fatalf("score %d exceeds the maximum of 100", score)
	}
}

func TestIsRoundAmount(t *testing.T) {
	cases := map[uint32]bool{
		0:     false,
		1:     false,
		99:    false,
		100:   true,
		500:   true,
		1234:  false,
		50000: true,
	}
	for amount, want := range cases {
		if got := isRoundAmount(amount); got != want {
			t.Errorf("isRoundAmount(%d) = %v, want %v", amount, got, want)
		}
	}
}

func TestCollectRiskSignalsUsesRealHistory(t *testing.T) {
	svc := &PaymentService{
		transactions: make(map[uint64]*Transaction),
	}
	cardToken := "tok_card_1"
	issued := time.Now().Add(-30 * 24 * time.Hour)

	svc.transactions[1] = &Transaction{CardID: cardToken, Status: "approved", Amount: 1000, Country: "US", MerchantCategory: "5411"}
	svc.transactions[2] = &Transaction{CardID: cardToken, Status: "approved", Amount: 3000, Country: "US", MerchantCategory: "5411"}
	// A declined transaction must not count towards the history.
	svc.transactions[3] = &Transaction{CardID: cardToken, Status: "declined", Amount: 90000, Country: "KP", MerchantCategory: "7995"}
	// Another card's transaction must not leak in.
	svc.transactions[4] = &Transaction{CardID: "tok_card_2", Status: "approved", Amount: 7000, Country: "DE", MerchantCategory: "5999"}

	card := &CardToken{TokenID: cardToken, CreatedAt: uint64(issued.UnixMilli())}
	req := &AuthorizationRequest{Amount: 2000, Country: "US", MerchantCategory: "5411"}

	signals := svc.collectRiskSignals(cardToken, card, req, time.Now())

	if signals.FirstUse {
		t.Error("card has an approved history, so FirstUse should be false")
	}
	if signals.RecentTxCount != 2 {
		t.Errorf("RecentTxCount = %d, want 2 (declined and other-card records excluded)", signals.RecentTxCount)
	}
	if signals.MeanAmount != 2000 {
		t.Errorf("MeanAmount = %d, want 2000", signals.MeanAmount)
	}
	if signals.MaxAmount != 3000 {
		t.Errorf("MaxAmount = %d, want 3000", signals.MaxAmount)
	}
	if signals.DistinctCountries != 1 {
		t.Errorf("DistinctCountries = %d, want 1", signals.DistinctCountries)
	}
	if signals.NewMerchantCategory {
		t.Error("category 5411 is already in the history, so NewMerchantCategory should be false")
	}
	if signals.CardAgeHours < 700 {
		t.Errorf("CardAgeHours = %f, want roughly 720", signals.CardAgeHours)
	}
}

func TestCollectRiskSignalsMarksFirstUse(t *testing.T) {
	svc := &PaymentService{transactions: make(map[uint64]*Transaction)}
	card := &CardToken{TokenID: "tok_new", CreatedAt: uint64(time.Now().UnixMilli())}
	req := &AuthorizationRequest{Amount: 500, MerchantCategory: "5411"}

	signals := svc.collectRiskSignals("tok_new", card, req, time.Now())

	if !signals.FirstUse {
		t.Error("a card with no approved history must be reported as first use")
	}
	if !signals.NewMerchantCategory {
		t.Error("the first merchant category must be reported as new")
	}
	if signals.RecentTxCount != 0 {
		t.Errorf("RecentTxCount = %d, want 0", signals.RecentTxCount)
	}
}
