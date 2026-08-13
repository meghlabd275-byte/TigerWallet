package main

// utils.go — small shared helpers (JSON encoding, pagination, etc.).

import (
	"encoding/json"
	"strconv"
)

// jsonMarshalImpl marshals to JSON (separate name to avoid clashing with the
// jsonMarshal wrapper used by handlers for cache writes).
func jsonMarshalImpl(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// detailsJSON marshals a map to JSONB-compatible JSON; nil on failure.
func detailsJSON(m map[string]interface{}) []byte {
	if m == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(m)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// parseLimit parses a pagination limit with a sane default + cap.
func parseLimit(s string, def, max int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// parseOffset parses a pagination offset.
func parseOffset(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
