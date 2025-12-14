package validator

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// BlockedNetworks contains IP ranges that should not be accessible
var BlockedNetworks = []string{
	"127.0.0.0/8",    // Localhost
	"10.0.0.0/8",     // Private network
	"172.16.0.0/12",  // Private network
	"192.168.0.0/16", // Private network
	"169.254.0.0/16", // Link-local (AWS metadata service)
}

// IsBlockedIP checks if an IP address is in a blocked network range
func IsBlockedIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	for _, cidr := range BlockedNetworks {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// ValidateHTTPURI validates an HTTP/HTTPS URI for SSRF prevention
func ValidateHTTPURI(uri string) error {
	parsed, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("invalid URI: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("expected http or https scheme")
	}

	hostname := parsed.Hostname()

	// Resolve hostname to IP addresses
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("failed to resolve hostname: %w", err)
	}

	// Check each resolved IP
	for _, ip := range ips {
		ipStr := ip.String()

		if IsBlockedIP(ipStr) {
			reason := getBlockReason(ipStr)
			return fmt.Errorf("access denied: %s resolves to %s (%s)", hostname, ipStr, reason)
		}
	}

	return nil
}

// getBlockReason returns a human-readable reason for blocking an IP
func getBlockReason(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "invalid IP"
	}

	if ip.IsLoopback() || strings.HasPrefix(ipStr, "127.") {
		return "localhost access not allowed"
	}

	for _, cidr := range BlockedNetworks {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			if strings.HasPrefix(cidr, "10.") || strings.HasPrefix(cidr, "172.16") || strings.HasPrefix(cidr, "192.168") {
				return "private network access not allowed"
			}
			if strings.HasPrefix(cidr, "169.254") {
				return "link-local access not allowed"
			}
		}
	}

	return "blocked network"
}
