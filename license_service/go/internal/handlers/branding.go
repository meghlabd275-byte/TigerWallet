package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/tigerwallet/license-service/internal/store"
)

// BrandingResponse is the PUBLIC payload returned by GET /api/v1/branding/:slug.
// It is intentionally a thin, projection view of the wl_clients.branding JSONB
// field plus the slug — never the full WLClient (which carries status, tier,
// contact email, etc. that are none of an app's business). Every field is
// optional; absent values fall back to TigerWallet defaults on the client.
type BrandingResponse struct {
	Slug           string `json:"slug"`
	AppName        string `json:"app_name"`
	LogoURL        string `json:"logo_url"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	Domain         string `json:"domain"`
	SupportEmail   string `json:"support_email"`
	TermsURL       string `json:"terms_url"`
	PrivacyURL     string `json:"privacy_url"`
}

// GetBrandingBySlug is a PUBLIC (no auth) endpoint. A WL-branded app calls it
// on startup with its build-time-injected slug to fetch its branding config.
// If no WL client matches the slug, or the slug is empty, we return 404 so the
// app cleanly falls back to TigerWallet defaults — no error UI, no crash.
func (h *Handlers) GetBrandingBySlug(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	if slug == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "branding not found"})
		return
	}
	ctx := c.Request.Context()
	wlc, err := h.store.GetWLClientBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "branding not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	// A suspended/halted/revoked WL client should not be serving a branded UX.
	// Fall back to defaults on the client by returning 404.
	if wlc.Status != "approved" && wlc.Status != "active" {
		c.JSON(http.StatusNotFound, gin.H{"error": "branding not found"})
		return
	}
	c.JSON(http.StatusOK, brandingFromClient(wlc))
}

// brandingFromClient projects a WLClient + its branding JSONB into the public
// BrandingResponse. Missing branding keys yield empty strings (app falls back).
func brandingFromClient(wlc *store.WLClient) BrandingResponse {
	b := BrandingResponse{Slug: wlc.Slug}
	if wlc.Branding == nil {
		return b
	}
	b.AppName = strVal(wlc.Branding, "app_name")
	b.LogoURL = strVal(wlc.Branding, "logo_url")
	b.PrimaryColor = strVal(wlc.Branding, "primary_color")
	b.SecondaryColor = strVal(wlc.Branding, "secondary_color")
	b.Domain = strVal(wlc.Branding, "domain")
	b.SupportEmail = strVal(wlc.Branding, "support_email")
	b.TermsURL = strVal(wlc.Branding, "terms_url")
	b.PrivacyURL = strVal(wlc.Branding, "privacy_url")
	return b
}

func strVal(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
