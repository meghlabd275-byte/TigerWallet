package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// A fresh deployment has no feature-flag keys in Redis. Builtin features must
// still work: the flag layer defaults to enabled and only an explicit operator
// decision gates a feature. Before this change a missing key resolved to
// disabled, so every lending call returned 423 until an admin toggled it.
func TestFeatureFlagDefaultsEnabledWithoutRedis(t *testing.T) {
	ls := &LendingService{}
	if got := ls.FeatureState(FeatureLending); got != flagStateEnabled {
		t.Fatalf("unset flag resolved to %q, want %q", got, flagStateEnabled)
	}
	if !ls.IsFeatureEnabled(FeatureLending) {
		t.Fatal("unset flag should be enabled")
	}
}

// The empty-name guard stays fail-closed: an unnamed feature is never enabled.
func TestFeatureFlagEmptyNameDisabled(t *testing.T) {
	ls := &LendingService{}
	if got := ls.FeatureState(""); got != flagStateDisabled {
		t.Fatalf("empty flag name resolved to %q, want %q", got, flagStateDisabled)
	}
}

func TestEnforceFeatureAllowsDefaultOn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ls := &LendingService{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if !ls.enforceFeature(c, FeatureLending) {
		t.Fatalf("enforceFeature blocked an unset flag with status %d", w.Code)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status written: %d", w.Code)
	}
}
