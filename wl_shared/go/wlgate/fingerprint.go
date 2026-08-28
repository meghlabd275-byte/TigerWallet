// Machine fingerprint for the no-resale instance binding.
//
// The fingerprint is a stable machine identity derived from local sysfs/DMI
// sources — it is NOT the license instance_id (which identifies deployment)
// and never contains any secret. It lets the control plane detect one license
// silently jumping to a different physical machine: the fingerprint changes
// even if the operator reuses the same instance_id.
package wlgate

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"sync"
)

var (
	fpOnce  sync.Once
	fpCache string
)

// Fingerprint returns a stable, hash-of-machine-id fingerprint for this host,
// or "" when the machine cannot be identified (containers without hardware
// identity fall back to their instance_id/client-provided id, so the gate's
// behavior is unchanged there).
func Fingerprint() string {
	fpOnce.Do(func() {
		var raw string
		for _, path := range []string{
			"/etc/machine-id",
			"/var/lib/dbus/machine-id",
		} {
			if b, err := os.ReadFile(path); err == nil {
				raw = strings.TrimSpace(string(b))
				break
			}
		}
		if raw == "" {
			if b, err := os.ReadFile("/sys/class/dmi/id/product_uuid"); err == nil {
				raw = "dmi:" + strings.ToLower(strings.TrimSpace(string(b)))
			}
		}
		if raw == "" {
			fpCache = ""
			return
		}
		sum := sha256.Sum256([]byte("tigerwallet-wl-fp-v1:" + raw))
		fpCache = hex.EncodeToString(sum[:16]) // 128-bit is plenty
	})
	return fpCache
}

// Hostname returns the machine hostname (best-effort; "" if unavailable).
func Hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}
