package main

import (
	"math"
	"time"
)

// Risk weights are expressed in points out of 100. A score at or above
// riskDeclineThreshold declines the authorization; a score at or above
// riskReviewThreshold approves it but flags it for manual review.
const (
	riskDeclineThreshold = 80
	riskReviewThreshold  = 50

	// A single authorization above this share of the card's per-transaction
	// limit is treated as high utilisation.
	highUtilisationRatio = 0.8
)

// riskSignals captures the per-card history the scorer needs. Keeping it a
// value type makes the scorer pure and straightforward to test.
type riskSignals struct {
	// Authorizations already seen for this card.
	RecentTxCount int
	// Mean amount of those authorizations, used to spot outliers.
	MeanAmount uint32
	// Largest amount previously authorized on this card.
	MaxAmount uint32
	// Number of distinct countries previously seen for this card.
	DistinctCountries int
	// True when this is the card's first authorization.
	FirstUse bool
	// True when the merchant category has not been seen before.
	NewMerchantCategory bool
	// True when the amount is a round number, which is typical of card testing.
	RoundAmount bool
	// Hours since the card was issued. Newly issued cards carry more risk.
	CardAgeHours float64
}

// calculateRiskScore produces a 0-100 risk score for a card authorization.
//
// The score is additive across independent signals and capped at 100. Each
// signal contributes a fixed weight so the outcome is reproducible and can be
// explained to an operator; nothing here is randomised or hardcoded per
// request.
func calculateRiskScore(req *AuthorizationRequest, limits *CardLimits, signals riskSignals) uint8 {
	var score float64

	// Amount relative to the card's own history. A transaction many times the
	// usual size is the strongest single indicator of account takeover.
	if signals.RecentTxCount > 0 && signals.MeanAmount > 0 {
		ratio := float64(req.Amount) / float64(signals.MeanAmount)
		switch {
		case ratio >= 10:
			score += 35
		case ratio >= 5:
			score += 25
		case ratio >= 3:
			score += 15
		case ratio >= 2:
			score += 8
		}
	}

	// Amount relative to the card's per-transaction limit.
	if limits != nil && limits.PerTransactionLimit > 0 {
		utilisation := float64(req.Amount) / float64(limits.PerTransactionLimit)
		switch {
		case utilisation >= 1:
			// Spending the entire per-transaction limit in a single attempt
			// is the strongest limit-based signal.
			score += 25
		case utilisation >= highUtilisationRatio:
			score += 12
		case utilisation >= 0.5:
			score += 6
		}
	}

	// A brand-new card used immediately is a common fraud pattern.
	if signals.CardAgeHours < 24 {
		score += 20
	} else if signals.CardAgeHours < 72 {
		score += 10
	}

	// First use on a card: some risk, but not enough to block on its own.
	if signals.FirstUse {
		score += 10
	}

	// An unseen merchant category suggests the card has been repurposed.
	if signals.NewMerchantCategory {
		score += 8
	}

	// Round amounts are characteristic of automated card testing.
	if signals.RoundAmount && req.Amount >= 100 {
		score += 12
	}

	// A country never seen for this card, once the card has an established
	// pattern, indicates the credentials have been used elsewhere.
	if signals.DistinctCountries > 0 && signals.DistinctCountries < 2 && !signals.FirstUse {
		score += 10
	}

	// A missing terminal identifier means the request did not come from a
	// recognised point-of-sale integration.
	if req.TerminalID == "" {
		score += 5
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return uint8(math.Round(score))
}

// isRoundAmount reports whether an amount is a round number of hundreds or
// thousands, which automated card testing tends to produce.
func isRoundAmount(amount uint32) bool {
	if amount == 0 {
		return false
	}
	return amount%1000 == 0 || amount%100 == 0
}

// collectRiskSignals derives the scoring inputs from the service's history for
// the card. All values come from records the service has actually observed.
func (s *PaymentService) collectRiskSignals(cardToken string, card *CardToken, req *AuthorizationRequest, now time.Time) riskSignals {
	signals := riskSignals{
		RoundAmount: isRoundAmount(req.Amount),
	}

	if card.CreatedAt > 0 {
		issued := time.UnixMilli(int64(card.CreatedAt))
		signals.CardAgeHours = now.Sub(issued).Hours()
	} else {
		// An unknown issue time is treated as new rather than trusted.
		signals.CardAgeHours = 0
	}

	// The card's issue time predates every authorization, so an empty history
	// means this is the first use.
	first := true
	var total uint64
	countries := make(map[string]struct{})
	categories := make(map[string]struct{})

	for _, tx := range s.transactions {
		if tx.CardID != cardToken {
			continue
		}
		if tx.Status != "approved" && tx.Status != "completed" {
			continue
		}
		first = false
		signals.RecentTxCount++
		total += uint64(tx.Amount)
		if tx.Amount > signals.MaxAmount {
			signals.MaxAmount = tx.Amount
		}
		if tx.Country != "" {
			countries[tx.Country] = struct{}{}
		}
		if tx.MerchantCategory != "" {
			categories[tx.MerchantCategory] = struct{}{}
		}
	}

	signals.FirstUse = first
	if signals.RecentTxCount > 0 {
		signals.MeanAmount = uint32(total / uint64(signals.RecentTxCount))
	}
	signals.DistinctCountries = len(countries)

	if req.MerchantCategory != "" {
		if _, seen := categories[req.MerchantCategory]; !seen {
			signals.NewMerchantCategory = true
		}
	}

	return signals
}
