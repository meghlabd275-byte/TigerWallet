// Package wlgate provides a Go binding to the C++ WlGate hot-path checker
// (wl_control_plane/cpp). WL products built in Go (master_wallet/backend,
// mm_bot_platform/bot_api, project_party/go, the standalone wl_user_wallet
// backend) link this and wrap every protected route in the Gate middleware,
// which fail-closeds (503) when the product is not alive or a fetcher is
// disabled by SuperAdmin.
//
// The real liveness + flag snapshot is established by the Rust SDK
// (white_level_sdk/rust), which a WL product boots in-process (via a tiny
// cgo shim) and calls wl_gate_set_alive / wl_gate_set_flags after each
// validate/heartbeat. This package exposes the C ABI symbols.
package wlgate

/*
#cgo CFLAGS: -I${SRCDIR}/../../cpp/include
#cgo LDFLAGS: -L${SRCDIR}/../../cpp/build -lwl_gate -lstdc++ -lm
#include "wl_gate_abi.h"
*/
import "C"

import (
	"encoding/json"
	"net/http"
	"strings"
	"unsafe"

	"github.com/gin-gonic/gin"
)

// IsAlive returns whether the product is licensed + heartbeat current.
func IsAlive() bool { return C.wl_gate_is_alive() == 1 }

// Reason returns the fail-closed reason (empty if alive).
func Reason() string {
	cstr := C.wl_gate_reason()
	if cstr == nil {
		return ""
	}
	return C.GoString(cstr)
}

// SetAlive sets the liveness flag (called by the Rust SDK bridge).
func SetAlive(alive bool, reason string) {
	var cr *C.char
	if reason != "" {
		cr = C.CString(reason)
		defer C.free(unsafe.Pointer(cr))
	}
	v := 0
	if alive {
		v = 1
	}
	C.wl_gate_set_alive(C.int(v), cr)
}

// SetFlags pushes a flag snapshot (JSON array) into the C++ gate.
func SetFlags(flagsJSON string) {
	cs := C.CString(flagsJSON)
	defer C.free(unsafe.Pointer(cs))
	C.wl_gate_set_flags(cs)
}

// SetFlagsFromStruct pushes a Go flag slice.
type Flag struct {
	Product string `json:"product"`
	Fetcher string `json:"fetcher"`
	Enabled bool   `json:"enabled"`
}

func SetFlagsFromStruct(flags []Flag) {
	b, _ := json.Marshal(flags)
	SetFlags(string(b))
}

// FetcherEnabled checks the per-fetcher gate.
func FetcherEnabled(product, fetcher string) bool {
	cp := C.CString(product)
	defer C.free(unsafe.Pointer(cp))
	cf := C.CString(fetcher)
	defer C.free(unsafe.Pointer(cf))
	return C.wl_gate_fetcher_enabled(cp, cf) == 1
}

// Gate is the gin middleware that fail-closeds every protected request. The
// product (e.g. "user_wallet") and a function deriving the fetcher name from
// the request path are provided. When the gate is closed, returns 503.
func Gate(product string, fetcherForPath func(path string) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsAlive() {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error":  "product is not authorized to serve (license suspended/revoked or heartbeat stale)",
				"reason": Reason(),
			})
			return
		}
		fetcher := "*"
		if fetcherForPath != nil {
			if f := fetcherForPath(c.Request.URL.Path); f != "" {
				fetcher = f
			}
		}
		if !FetcherEnabled(product, fetcher) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "fetcher disabled by SuperAdmin",
				"product": product,
				"fetcher": fetcher,
			})
			return
		}
		c.Next()
	}
}

// SimpleFetcher derives the fetcher name from the last path segment. Many WL
// routes map 1:1 (e.g. /balance -> "balance"); for routes where a disabled
// sub-tree should block, the product supplies a custom mapper.
func SimpleFetcher(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return "*"
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
