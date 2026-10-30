package main

// persistence.go — Redis-backed indexes for claims and alerts.
//
// Claims and alerts are stored under their own keys. These helpers maintain the
// secondary indexes that let a user's records be listed, so the list endpoints
// return what was actually recorded rather than an empty result.

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-redis/redis/v8"
)

// Index TTLs. Claim history is retained for the life of a policy; alerts are
// operational and expire sooner.
const (
	claimIndexTTL = 365 * 24 * time.Hour
	alertIndexTTL = 30 * 24 * time.Hour
)

// userClaimsKey is the sorted set holding a user's claim ids, scored by
// creation time.
func userClaimsKey(userID string) string {
	return fmt.Sprintf("claims:user:%s", userID)
}

// userAlertsKey is the sorted set holding a user's alert ids, scored by
// creation time.
func userAlertsKey(userID string) string {
	return fmt.Sprintf("alerts:user:%s", userID)
}

// claimKeyFor returns the key holding a claim document.
func claimKeyFor(id string) string {
	return fmt.Sprintf("claim:%s", id)
}

// alertKeyFor returns the key holding an alert document.
func alertKeyFor(id string) string {
	return fmt.Sprintf("alert:%s", id)
}

// saveClaim stores a claim and adds it to the owner's index.
func saveClaim(ctx context.Context, client *redis.Client, claim Claim) error {
	data, err := json.Marshal(claim)
	if err != nil {
		return fmt.Errorf("encode claim: %w", err)
	}

	pipe := client.TxPipeline()
	pipe.Set(ctx, claimKeyFor(claim.ClaimID), data, claimIndexTTL)
	pipe.ZAdd(ctx, userClaimsKey(claim.UserID), &redis.Z{
		Score:  float64(claim.CreatedAt),
		Member: claim.ClaimID,
	})
	pipe.Expire(ctx, userClaimsKey(claim.UserID), claimIndexTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("store claim: %w", err)
	}
	return nil
}

// loadClaim reads a claim by id.
func loadClaim(ctx context.Context, client *redis.Client, id string) (Claim, error) {
	data, err := client.Get(ctx, claimKeyFor(id)).Bytes()
	if err != nil {
		return Claim{}, err
	}
	var claim Claim
	if err := json.Unmarshal(data, &claim); err != nil {
		return Claim{}, fmt.Errorf("decode claim: %w", err)
	}
	return claim, nil
}

// listUserClaims returns a user's claims, newest first.
//
// Ids whose documents have expired are skipped rather than returned as empty
// records, and the index is repaired so the stale id is not retried.
func listUserClaims(ctx context.Context, client *redis.Client, userID string) ([]Claim, error) {
	ids, err := client.ZRevRange(ctx, userClaimsKey(userID), 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return []Claim{}, nil
		}
		return nil, fmt.Errorf("list claims: %w", err)
	}
	if len(ids) == 0 {
		return []Claim{}, nil
	}

	claims := make([]Claim, 0, len(ids))
	var stale []interface{}
	for _, id := range ids {
		claim, err := loadClaim(ctx, client, id)
		if err != nil {
			stale = append(stale, id)
			continue
		}
		claims = append(claims, claim)
	}
	if len(stale) > 0 {
		client.ZRem(ctx, userClaimsKey(userID), stale...)
	}
	return claims, nil
}

// saveAlert stores an alert and adds it to the owner's index.
func saveAlert(ctx context.Context, client *redis.Client, alert *TransactionAlert) error {
	data, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("encode alert: %w", err)
	}

	pipe := client.TxPipeline()
	pipe.Set(ctx, alertKeyFor(alert.AlertID), data, alertIndexTTL)
	pipe.ZAdd(ctx, userAlertsKey(alert.UserID), &redis.Z{
		Score:  float64(alert.CreatedAt),
		Member: alert.AlertID,
	})
	pipe.Expire(ctx, userAlertsKey(alert.UserID), alertIndexTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("store alert: %w", err)
	}
	return nil
}

// loadAlert reads an alert by id.
func loadAlert(ctx context.Context, client *redis.Client, id string) (*TransactionAlert, error) {
	data, err := client.Get(ctx, alertKeyFor(id)).Bytes()
	if err != nil {
		return nil, err
	}
	var alert TransactionAlert
	if err := json.Unmarshal(data, &alert); err != nil {
		return nil, fmt.Errorf("decode alert: %w", err)
	}
	return &alert, nil
}

// listUserAlerts returns a user's alerts, newest first.
func listUserAlerts(ctx context.Context, client *redis.Client, userID string) ([]*TransactionAlert, error) {
	ids, err := client.ZRevRange(ctx, userAlertsKey(userID), 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return []*TransactionAlert{}, nil
		}
		return nil, fmt.Errorf("list alerts: %w", err)
	}

	alerts := make([]*TransactionAlert, 0, len(ids))
	var stale []interface{}
	for _, id := range ids {
		alert, err := loadAlert(ctx, client, id)
		if err != nil {
			stale = append(stale, id)
			continue
		}
		alerts = append(alerts, alert)
	}
	if len(stale) > 0 {
		client.ZRem(ctx, userAlertsKey(userID), stale...)
	}
	return alerts, nil
}

// coverageSummary is the usage computed from a user's recorded claims.
type coverageSummary struct {
	TotalUsedUSD  float64
	ClaimsCount   int
	PendingCount  int
	ApprovedCount int
	RejectedCount int
	ClaimHistory  []Claim
}

// summarizeCoverage derives coverage usage from the claims actually recorded.
//
// The used amount counts claims that were approved or paid, because a pending
// claim has not consumed coverage yet. Previously this reported the
// policy's stored CoverageUsedUSD with a hardcoded claim count of zero, so the
// usage endpoint never reflected the claims a user had filed.
func summarizeCoverage(claims []Claim) coverageSummary {
	summary := coverageSummary{ClaimHistory: claims}
	for _, claim := range claims {
		summary.ClaimsCount++
		switch claim.Status {
		case "approved", "paid":
			summary.ApprovedCount++
			// An approved claim that carries an explicit approved amount uses
			// it; otherwise the requested amount was granted in full.
			if claim.ApprovedAmount > 0 {
				summary.TotalUsedUSD += claim.ApprovedAmount
			} else {
				summary.TotalUsedUSD += claim.AmountUSD
			}
		case "rejected":
			summary.RejectedCount++
		default:
			summary.PendingCount++
		}
	}
	// Newest first, so the history matches the order of the list endpoint.
	sort.SliceStable(summary.ClaimHistory, func(i, j int) bool {
		return summary.ClaimHistory[i].CreatedAt > summary.ClaimHistory[j].CreatedAt
	})
	return summary
}
