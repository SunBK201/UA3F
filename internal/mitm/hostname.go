package mitm

import (
	"fmt"
	"log/slog"
	"path"
	"strconv"
	"strings"
)

// HostnameEntry represents a parsed hostname rule for MitM filtering.
// Format: domain[:port]
//   - Default port is 443
//   - Port 0 means match all ports
//   - Domain supports standard glob patterns (e.g., *.example.com, api-?.example.com, [ab].example.com)
type HostnameEntry struct {
	Domain  string // glob pattern, e.g., "*.example.com", "api.example.com"
	Port    string // "0" = all ports, other = specific port (e.g., "443")
	AllPort bool   // true if port is "0" (match all ports)
}

// HostnameFilter decides whether a given hostname:port should be MitM'd.
type HostnameFilter struct {
	entries []HostnameEntry
}

// NewHostnameFilter parses a comma-separated hostname list and returns a filter.
// If the hostname string is empty, the filter will match no hostnames (i.e., MitM disabled).
// Domain patterns use standard glob syntax (see path.Match):
//   - *        matches any sequence of non-/ characters
//   - ?        matches any single character
//   - [abc]    matches any character in the set
//   - [a-z]    matches any character in the range
//
// Format examples:
//   - "example.com"              → match example.com on port 443
//   - "example.com:8443"         → match example.com on port 8443
//   - "example.com:0"            → match example.com on all ports
//   - "*.example.com"            → match any subdomain of example.com on port 443
//   - "*.example.com:0"          → match any subdomain of example.com on all ports
//   - "*"                        → match all domains on port 443
//   - "*:0"                      → match all domains on all ports
//   - "api-?.example.com"        → match api-X.example.com (single char) on port 443
func NewHostnameFilter(hostname string) (*HostnameFilter, error) {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		slog.Info("MitM hostname filter is empty, no hostnames will be intercepted")
		return &HostnameFilter{}, nil
	}

	parts := strings.Split(hostname, ",")
	entries := make([]HostnameEntry, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		entry, err := parseHostnameEntry(part)
		if err != nil {
			slog.Error("MitM hostname filter parse error", slog.String("entry", part), slog.Any("error", err))
			continue
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		slog.Info("MitM hostname filter is empty, no hostnames will be intercepted")
		return &HostnameFilter{}, nil
	}

	slog.Info("MitM hostname filter configured", slog.Int("entries", len(entries)))
	return &HostnameFilter{entries: entries}, nil
}

// parseHostnameEntry parses a single hostname entry like "*.example.com:443".
func parseHostnameEntry(s string) (HostnameEntry, error) {
	entry := HostnameEntry{
		Port: "443", // default port
	}

	// Check if there's a port suffix.
	// Be careful with IPv6 addresses, but SNI hostnames are always domain names.
	lastColon := strings.LastIndex(s, ":")
	if lastColon >= 0 {
		portStr := s[lastColon+1:]
		port, err := strconv.Atoi(portStr)
		if err == nil {
			// Valid port number found
			if port < 0 || port > 65535 {
				return entry, fmt.Errorf("port %d out of range (0-65535)", port)
			}
			entry.Domain = s[:lastColon]
			if port == 0 {
				entry.AllPort = true
				entry.Port = "0"
			} else {
				entry.Port = portStr
			}
		} else {
			// Not a valid port, treat entire string as domain
			entry.Domain = s
		}
	} else {
		entry.Domain = s
	}

	entry.Domain = strings.TrimSpace(entry.Domain)
	if entry.Domain == "" {
		return entry, fmt.Errorf("empty domain")
	}

	return entry, nil
}

// Allow checks whether the given serverName and port should be MitM'd.
// serverName is the SNI hostname from the TLS ClientHello.
// port is the destination port (e.g., 443).
func (f *HostnameFilter) Allow(serverName string, port string) bool {
	if f == nil {
		return false
	}

	for _, entry := range f.entries {
		if !entry.matchPort(port) {
			continue
		}
		if matchDomain(entry.Domain, serverName) {
			return true
		}
	}

	return false
}

// matchPort checks if the port matches this entry.
func (e *HostnameEntry) matchPort(port string) bool {
	if e.AllPort {
		return true
	}
	return e.Port == port
}

// matchDomain checks if serverName matches the domain pattern using standard glob matching.
// Uses path.Match glob syntax:
//   - "*" matches everything (any sequence of non-/ characters, and hostnames have no /)
//   - "*.example.com" matches "foo.example.com", "bar.example.com",
//     "a.b.example.com", but NOT "example.com" itself
//   - "api-?.example.com" matches "api-1.example.com", "api-a.example.com"
//   - "example.com" matches "example.com" exactly
func matchDomain(pattern, serverName string) bool {
	pattern = strings.ToLower(pattern)
	serverName = strings.ToLower(serverName)

	matched, err := path.Match(pattern, serverName)
	if err != nil {
		// Invalid glob pattern, fall back to exact match
		return pattern == serverName
	}
	return matched
}
