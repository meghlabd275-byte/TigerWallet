// feature_flags.go — downstream feature-flag enforcement layer (stdlib).
//
// Mirrors go/wallet_api/feature_flags.go. Redis is the SHARED feature-flag
// store. Admin backends WRITE flag state to Redis; downstream services (this
// one) READ it:
//
//	Key:   tigerwallet:feature:<name>
//	Value: "enabled" | "disabled" | "paused"
//
// Enforcement is fail-closed: missing/unknown/erroring state => disabled.
// This file implements a minimal Redis RESP GET client with only the stdlib
// so no new module dependency is required.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	flagStateEnabled  = "enabled"
	flagStateDisabled = "disabled"
	flagStatePaused   = "paused"
)

const flagKeyPrefix = "tigerwallet:feature:"

func flagKey(name string) string { return flagKeyPrefix + name }

// GatedFeature is the canonical feature-flag name for this service.
const GatedFeature = "red_packets"

const flagCacheTTL = 5 * time.Second

type flagCacheEntry struct {
	state     string
	fetchedAt time.Time
}

var (
	flagCacheMu sync.Mutex
	flagCache   = make(map[string]flagCacheEntry)
)

func redisAddr() string {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	return addr
}

// redisGet performs a single RESP GET and returns the bulk string value.
// Returns an error for nil (missing key), connection failure, or bad reply.
func redisGet(key string) (string, error) {
	conn, err := net.DialTimeout("tcp", redisAddr(), 500*time.Millisecond)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(500 * time.Millisecond))

	// *2\r\n$3\r\nGET\r\n$<len>\r\n<key>\r\n
	cmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key)
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return "", err
	}

	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "$-1" {
		return "", errors.New("nil")
	}
	if !strings.HasPrefix(line, "$") {
		return "", fmt.Errorf("unexpected reply %q", line)
	}
	n, err := strconv.Atoi(line[1:])
	if err != nil {
		return "", err
	}
	buf := make([]byte, n+2) // value + CRLF
	got := 0
	for got < n+2 {
		m, err := r.Read(buf[got:])
		if err != nil {
			return "", err
		}
		got += m
	}
	return string(buf[:n]), nil
}

func fetchFeatureState(featureName string) string {
	val, err := redisGet(flagKey(featureName))
	if err != nil {
		return flagStateDisabled
	}
	switch val {
	case flagStateEnabled, flagStateDisabled, flagStatePaused:
		return val
	default:
		return flagStateDisabled
	}
}

// FeatureState returns the cached-then-live state for the feature.
func FeatureState(featureName string) string {
	if featureName == "" {
		return flagStateDisabled
	}
	now := time.Now()
	flagCacheMu.Lock()
	if entry, ok := flagCache[featureName]; ok && now.Sub(entry.fetchedAt) < flagCacheTTL {
		flagCacheMu.Unlock()
		return entry.state
	}
	flagCacheMu.Unlock()
	state := fetchFeatureState(featureName)
	flagCacheMu.Lock()
	flagCache[featureName] = flagCacheEntry{state: state, fetchedAt: now}
	flagCacheMu.Unlock()
	return state
}

func isFeatureEnabled(featureName string) bool {
	return FeatureState(featureName) == flagStateEnabled
}

// enforceFeature writes HTTP 423 and returns false when the feature is off.
// Usage at the top of a handler:
//
//	if !enforceFeature(w, GatedFeature) { return }
func enforceFeature(w http.ResponseWriter, featureName string) bool {
	if isFeatureEnabled(featureName) {
		return true
	}
	state := FeatureState(featureName)
	if state == "" {
		state = flagStateDisabled
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusLocked)
	_, _ = w.Write([]byte(`{"error":"feature ` + featureName + ` is currently ` + state + `"}`))
	return false
}
