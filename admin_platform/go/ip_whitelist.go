package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// IP WHITELIST SERVICE
// ============================================================================

type IPWhitelistService struct {
	mu         sync.RWMutex
	whitelists map[string]map[string]*IPRange // adminID -> IP rules
}

type IPRange struct {
	StartIP  net.IP
	EndIP    net.IP
	Mask     net.IPMask
	CIDR     string
	IsRange  bool
	IsActive bool
}

func NewIPWhitelistService() *IPWhitelistService {
	return &IPWhitelistService{
		whitelists: make(map[string]map[string]*IPRange),
	}
}

// AddWhitelist adds an IP to the whitelist
func (s *IPWhitelistService) AddWhitelist(adminID, ipOrCIDR string) error {
	ipRange, err := s.parseIPOrCIDR(ipOrCIDR)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.whitelists[adminID]; !ok {
		s.whitelists[adminID] = make(map[string]*IPRange)
	}

	s.whitelists[adminID][ipOrCIDR] = ipRange
	return nil
}

// RemoveWhitelist removes an IP from the whitelist
func (s *IPWhitelistService) RemoveWhitelist(adminID, ipOrCIDR string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rules, ok := s.whitelists[adminID]; ok {
		delete(rules, ipOrCIDR)
	}
}

// IsAllowed checks if an IP is whitelisted
func (s *IPWhitelistService) IsAllowed(adminID, clientIP string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check specific admin whitelist
	if rules, ok := s.whitelists[adminID]; ok {
		for _, ipRange := range rules {
			if ipRange.IsActive && s.ipInRange(clientIP, ipRange) {
				return true
			}
		}
	}

	return false
}

// GetWhitelists returns all whitelisted IPs for an admin
func (s *IPWhitelistService) GetWhitelists(adminID string) []*IPRange {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if rules, ok := s.whitelists[adminID]; ok {
		result := make([]*IPRange, 0, len(rules))
		for _, r := range rules {
			result = append(result, r)
		}
		return result
	}

	return nil
}

func (s *IPWhitelistService) parseIPOrCIDR(ipOrCIDR string) (*IPRange, error) {
	ip := net.ParseIP(ipOrCIDR)
	if ip != nil {
		return &IPRange{
			StartIP:  ip,
			EndIP:    ip,
			IsRange:  false,
			IsActive: true,
		}, nil
	}

	// Try CIDR notation
	if !strings.Contains(ipOrCIDR, "/") {
		ipOrCIDR += "/32"
	}

	_, ipNet, err := net.ParseCIDR(ipOrCIDR)
	if err != nil {
		// Try as IP range (e.g., 192.168.1.1-192.168.1.255)
		if strings.Contains(ipOrCIDR, "-") {
			return s.parseIPRange(ipOrCIDR)
		}
		return nil, fmt.Errorf("invalid IP or CIDR: %w", err)
	}

	ip = ipNet.IP.To4()
	if ip == nil {
		ip = ipNet.IP.To16()
	}

	mask := ipNet.Mask
	startIP := ip.Mask(mask)
	broadcast := make(net.IP, len(ip))
	for i := range startIP {
		broadcast[i] = startIP[i] | ^mask[i]
	}

	return &IPRange{
		StartIP:  startIP,
		EndIP:    broadcast,
		Mask:     mask,
		CIDR:     ipOrCIDR,
		IsRange:  true,
		IsActive: true,
	}, nil
}

func (s *IPWhitelistService) parseIPRange(ipRange string) (*IPRange, error) {
	parts := strings.Split(ipRange, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid IP range format")
	}

	startIP := net.ParseIP(strings.TrimSpace(parts[0]))
	if startIP == nil {
		return nil, fmt.Errorf("invalid start IP")
	}

	endIPStr := strings.TrimSpace(parts[1])
	endIP := net.ParseIP(endIPStr)

	// If end IP is just the last octet
	if endIP == nil {
		startIPParts := strings.Split(startIP.String(), ".")
		if len(startIPParts) == 4 {
			endIP = net.ParseIP(startIPParts[0] + "." + startIPParts[1] + "." + startIPParts[2] + "." + endIPStr)
		}
	}

	if endIP == nil {
		return nil, fmt.Errorf("invalid end IP")
	}

	return &IPRange{
		StartIP:  startIP,
		EndIP:    endIP,
		IsRange:  true,
		IsActive: true,
	}, nil
}

func (s *IPWhitelistService) ipInRange(clientIP string, ipRange *IPRange) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	// Handle IPv4 vs IPv6
	if ipRange.StartIP.To4() != nil && ip.To4() == nil {
		return false
	}

	for i := range ip {
		if i < len(ipRange.StartIP) && i < len(ip) {
			if ip[i] < ipRange.StartIP[i] || ip[i] > ipRange.EndIP[i] {
				return false
			}
		}
	}

	return true
}

// ============================================================================
// IP WHITELIST MIDDLEWARE
// ============================================================================

func (s *IPWhitelistService) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip for health checks
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		// Skip for auth endpoints (with rate limiting)
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/auth/") {
			c.Next()
			return
		}

		// Get admin ID from context
		adminID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
			clientIP = strings.Split(forwarded, ",")[0]
		}

		// Check if IP is whitelisted
		if !s.IsAllowed(adminID.(string), clientIP) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":     "Access denied",
				"message":   "Your IP address is not whitelisted",
				"whitelist": true,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func isValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

// GetClientIP extracts the real client IP from request
func GetClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header
	if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fall back to direct client IP
	return c.ClientIP()
}

// ============================================================================
// GEOLOCATION SERVICE
// ============================================================================

type GeoLocation struct {
	Country   string `json:"country"`
	City      string `json:"city"`
	Region    string `json:"region"`
	ISP       string `json:"isp"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

type GeoIPService struct {
	// In production, use MaxMind GeoIP database or similar service
}

func NewGeoIPService() *GeoIPService {
	return &GeoIPService{}
}

// Lookup looks up geolocation for an IP address
func (g *GeoIPService) Lookup(ip string) (*GeoLocation, error) {
	// In production, use MaxMind GeoIP database
	// For now, return mock data
	return &GeoLocation{
		Country:   "US",
		City:      "San Francisco",
		Region:    "CA",
		ISP:       "Example ISP",
		Latitude:  "37.7749",
		Longitude: "-122.4194",
	}, nil
}

// IsHighRisk checks if an IP is from a high-risk location
func (g *GeoIPService) IsHighRisk(ip string) bool {
	geo, err := g.Lookup(ip)
	if err != nil {
		return true // Assume high risk if can't determine
	}

	// List of high-risk countries (example)
	highRiskCountries := map[string]bool{
		"NK": true, // North Korea
		"IR": true, // Iran
		"SY": true, // Syria
		"CU": true, // Cuba
	}

	return highRiskCountries[geo.Country]
}
