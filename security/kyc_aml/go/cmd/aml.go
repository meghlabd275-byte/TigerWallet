// AML screening.
//
// Screening runs against a configured provider and the operator's own watchlist
// database. A screening that cannot be completed is reported as failed rather
// than as a pass, so a provider outage never silently clears a customer.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// amlScreener screens a person against sanctions, PEP and adverse-media lists.
type amlScreener struct {
	store    *Store
	provider *AMLProviderConfig
	client   *http.Client
}

// AMLProviderConfig configures the external screening provider.
type AMLProviderConfig struct {
	Endpoint string
	APIKey   string
}

// screeningResult is the outcome of one screening run.
type screeningResult struct {
	PepMatch          bool
	SanctionMatch     bool
	AdverseMediaMatch bool
	RiskScore         int
	RiskLevel         string
	Details           map[string]interface{}
}

// newAMLScreener builds a screener over the given store.
func newAMLScreener(store *Store, cfg *AMLProviderConfig) *amlScreener {
	return &amlScreener{
		store:    store,
		provider: cfg,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// screen runs the provider screening, adds any watchlist matches held in this
// database, and computes the final risk score.
func (a *amlScreener) screen(ctx context.Context, firstName, lastName, country string) (*screeningResult, error) {
	result := &screeningResult{Details: map[string]interface{}{}}

	if a.provider != nil && strings.TrimSpace(a.provider.Endpoint) != "" {
		if !strings.HasPrefix(a.provider.Endpoint, "https://") {
			return nil, fmt.Errorf("AML provider endpoint must use https")
		}

		payload, err := json.Marshal(map[string]string{
			"first_name": firstName,
			"last_name":  lastName,
			"country":    country,
		})
		if err != nil {
			return nil, err
		}

		reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, a.provider.Endpoint, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if a.provider.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+a.provider.APIKey)
		}

		resp, err := a.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("AML provider call failed: %w", err)
		}
		defer resp.Body.Close()

		raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return nil, fmt.Errorf("read AML provider response: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("AML provider returned HTTP %d: %s", resp.StatusCode, truncateForLog(string(raw), 256))
		}

		var decoded struct {
			PepMatch          bool `json:"pep_match"`
			SanctionMatch     bool `json:"sanction_match"`
			AdverseMediaMatch bool `json:"adverse_media_match"`
			RiskScore         int  `json:"risk_score"`
		}
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, fmt.Errorf("decode AML provider response: %w", err)
		}

		result.PepMatch = decoded.PepMatch
		result.SanctionMatch = decoded.SanctionMatch
		result.AdverseMediaMatch = decoded.AdverseMediaMatch
		result.RiskScore = decoded.RiskScore
		result.Details["provider"] = "external"
	} else {
		result.Details["provider"] = "watchlist-only"
	}

	// Match the applicant against this operator's own watchlist.
	fullName := strings.TrimSpace(firstName + " " + lastName)
	if fullName != "" {
		matches, err := a.store.SearchWatchlist(ctx, fullName, 10)
		if err != nil {
			return nil, fmt.Errorf("watchlist lookup failed: %w", err)
		}
		if len(matches) > 0 {
			names := make([]string, 0, len(matches))
			for _, m := range matches {
				names = append(names, m.Name)
				switch strings.ToLower(m.ListType) {
				case "sanctions":
					result.SanctionMatch = true
				case "pep":
					result.PepMatch = true
				}
				if strings.EqualFold(m.RiskLevel, "high") {
					result.AdverseMediaMatch = true
				}
			}
			result.Details["watchlist_matches"] = names
		}
	}

	result.RiskScore = scoreRisk(result)
	result.RiskLevel = riskLevelFor(result.RiskScore)
	return result, nil
}

// scoreRisk converts matches into a bounded risk score.
func scoreRisk(result *screeningResult) int {
	score := 0
	if result.SanctionMatch {
		score += 100
	}
	if result.PepMatch {
		score += 50
	}
	if result.AdverseMediaMatch {
		score += 30
	}
	score += result.RiskScore
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

// riskLevelFor maps a score to a level. A sanctions match is always high.
func riskLevelFor(score int) string {
	switch {
	case score >= 70:
		return "high"
	case score >= 30:
		return "medium"
	default:
		return "low"
	}
}

// runAMLCheck screens a person, stores the result and returns it.
func (a *amlScreener) runAMLCheck(ctx context.Context, userID, applicationID uuid.UUID, firstName, lastName, country string) (*AMLCheck, error) {
	result, err := a.screen(ctx, firstName, lastName, country)
	if err != nil {
		// A screening that did not run is recorded as failed so the applicant
		// is not treated as cleared.
		check := &AMLCheck{
			ID:            uuid.New(),
			UserID:        userID,
			ApplicationID: applicationID,
			Status:        "failed",
			RiskScore:     100,
			RiskLevel:     "high",
			Details:       map[string]interface{}{"error": err.Error()},
			CheckedAt:     time.Now(),
			CreatedAt:     time.Now(),
		}
		if storeErr := a.store.CreateAMLCheck(ctx, check); storeErr != nil {
			log.Printf("could not record failed AML check: %v", storeErr)
		}
		if appErr := a.store.SetAMLCheckResult(ctx, applicationID, false); appErr != nil && appErr != ErrNotFound {
			log.Printf("could not flag application %s as unscreened: %v", applicationID, appErr)
		}
		return check, nil
	}

	passed := !result.SanctionMatch && result.RiskScore < 70

	check := &AMLCheck{
		ID:                uuid.New(),
		UserID:            userID,
		ApplicationID:     applicationID,
		Status:            "completed",
		PepMatch:          result.PepMatch,
		SanctionMatch:     result.SanctionMatch,
		AdverseMediaMatch: result.AdverseMediaMatch,
		RiskScore:         result.RiskScore,
		RiskLevel:         result.RiskLevel,
		Details:           result.Details,
		CheckedAt:         time.Now(),
		CreatedAt:         time.Now(),
	}

	if err := a.store.CreateAMLCheck(ctx, check); err != nil {
		return nil, err
	}
	if err := a.store.SetAMLCheckResult(ctx, applicationID, passed); err != nil && err != ErrNotFound {
		return nil, err
	}
	return check, nil
}

// truncateForLog shortens a message for an error string.
func truncateForLog(s string, limit int) string {
	s = strings.TrimSpace(s)
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
