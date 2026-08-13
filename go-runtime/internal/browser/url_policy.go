package browser

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// URLFirewall applies the canonical network allowlist and SSRF protections.
type URLFirewall struct {
	root   string
	lookup func(context.Context, string) ([]net.IP, error)
}

// NewURLFirewall creates a firewall backed by .ovav/registry/network_allowlist.yaml.
func NewURLFirewall(root string) *URLFirewall {
	return &URLFirewall{root: root, lookup: func(ctx context.Context, host string) ([]net.IP, error) {
		return net.DefaultResolver.LookupIP(ctx, "ip", host)
	}}
}

// Validate rejects unsafe URL schemes, credentials, non-allowlisted domains,
// and addresses that resolve to private networks. Explicit loopback URLs are
// allowed for local browser and CDP operation.
func (firewall *URLFirewall) Validate(target string) error {
	parsed, err := url.Parse(target)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return fmt.Errorf("browser URL denied: only absolute http(s) URLs are allowed")
	}
	if parsed.User != nil {
		return fmt.Errorf("browser URL denied: credentials are prohibited")
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() {
			return nil
		}
		if unsafeNetworkIP(ip) {
			return fmt.Errorf("browser URL denied: private-network targets are prohibited")
		}
	} else if host == "localhost" {
		return nil
	}

	patterns, err := loadNetworkAllowlist(firewall.root)
	if err != nil {
		return fmt.Errorf("browser URL denied: network allowlist unavailable: %w", err)
	}
	allowed := false
	for _, pattern := range patterns {
		if host == pattern || (strings.HasPrefix(pattern, "*.") && strings.HasSuffix(host, pattern[1:])) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("browser URL denied: domain is not allowlisted")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	addresses, err := firewall.lookup(ctx, host)
	if err != nil || len(addresses) == 0 {
		return fmt.Errorf("browser URL denied: domain resolution failed")
	}
	for _, address := range addresses {
		if unsafeNetworkIP(address) {
			return fmt.Errorf("browser URL denied: domain resolves to a private network")
		}
	}
	return nil
}

func unsafeNetworkIP(ip net.IP) bool {
	return ip.IsPrivate() || ip.IsUnspecified() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

func loadNetworkAllowlist(root string) ([]string, error) {
	path := filepath.Join(root, ".ovav", "registry", "network_allowlist.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- pattern:") {
			continue
		}
		value := strings.ToLower(strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "- pattern:")), `"'`))
		if value != "" && value != "*" {
			patterns = append(patterns, value)
		}
	}
	if len(patterns) == 0 {
		return nil, errors.New("no active domain patterns")
	}
	return patterns, nil
}
