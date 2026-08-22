// TigerWallet Launchpad - legacy project model.
//
// This file previously declared package-level duplicates of Config, getEnv,
// Allocation, LaunchpadService and a main() that conflicted with the
// canonical implementations in launchpad_service.go. Those duplicates have
// been removed; only the distinct, redis-backed Project model that this file
// uniquely defined is retained here for reference/back-compat consumers.

package launchpad

import "time"

// Project is the legacy redis-backed launchpad project representation kept
// for compatibility. The canonical, DB-backed model is LaunchpadProject in
// launchpad_service.go.
type Project struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Token         string    `json:"token"`
	TokenAddress  string    `json:"tokenAddress"`
	SoftCap       float64   `json:"softCap"`
	HardCap       float64   `json:"hardCap"`
	MinBuy        float64   `json:"minBuy"`
	MaxBuy        float64   `json:"maxBuy"`
	Price         float64   `json:"price"`
	StartTime     time.Time `json:"startTime"`
	EndTime       time.Time `json:"endTime"`
	Status        string    `json:"status"`
	TotalRaised   float64   `json:"totalRaised"`
	Participants  int       `json:"participants"`
	Logo          string    `json:"logo"`
	Website       string    `json:"website"`
	Whitepaper    string    `json:"whitepaper"`
}
