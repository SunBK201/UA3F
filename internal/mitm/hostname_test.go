package mitm

import (
	"testing"
)

func TestParseHostnameEntry(t *testing.T) {
	tests := []struct {
		input   string
		domain  string
		port    string
		allPort bool
		wantErr bool
	}{
		{"example.com", "example.com", "443", false, false},
		{"example.com:8443", "example.com", "8443", false, false},
		{"example.com:0", "example.com", "0", true, false},
		{"*.example.com", "*.example.com", "443", false, false},
		{"*.example.com:0", "*.example.com", "0", true, false},
		{"*.example.com:8443", "*.example.com", "8443", false, false},
		{"*", "*", "443", false, false},
		{"*:0", "*", "0", true, false},
		{":443", "", "", false, true},              // empty domain
		{"example.com:99999", "", "", false, true}, // port out of range
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			entry, err := parseHostnameEntry(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got none", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if entry.Domain != tt.domain {
				t.Errorf("domain: got %q, want %q", entry.Domain, tt.domain)
			}
			if entry.Port != tt.port {
				t.Errorf("port: got %s, want %s", entry.Port, tt.port)
			}
			if entry.AllPort != tt.allPort {
				t.Errorf("allPort: got %v, want %v", entry.AllPort, tt.allPort)
			}
		})
	}
}

func TestMatchDomain(t *testing.T) {
	tests := []struct {
		pattern    string
		serverName string
		want       bool
	}{
		{"example.com", "example.com", true},
		{"example.com", "Example.COM", true},
		{"example.com", "foo.example.com", false},
		{"*.example.com", "foo.example.com", true},
		{"*.example.com", "bar.example.com", true},
		{"*.example.com", "example.com", false},
		{"*.example.com", "foo.bar.example.com", true},
		{"*", "anything.com", true},
		{"*", "foo.bar.baz", true},
		{"api.example.com", "api.example.com", true},
		{"api.example.com", "other.example.com", false},

		// Glob: ? matches single character
		{"api-?.example.com", "api-1.example.com", true},
		{"api-?.example.com", "api-a.example.com", true},
		{"api-?.example.com", "api-ab.example.com", false},

		// Glob: [...] character class
		{"[ab].example.com", "a.example.com", true},
		{"[ab].example.com", "b.example.com", true},
		{"[ab].example.com", "c.example.com", false},

		// Glob: [a-z] character range
		{"api-[0-9].example.com", "api-5.example.com", true},
		{"api-[0-9].example.com", "api-a.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.serverName, func(t *testing.T) {
			got := matchDomain(tt.pattern, tt.serverName)
			if got != tt.want {
				t.Errorf("matchDomain(%q, %q) = %v, want %v", tt.pattern, tt.serverName, got, tt.want)
			}
		})
	}
}

func TestHostnameFilterAllow(t *testing.T) {
	tests := []struct {
		name       string
		hostname   string
		serverName string
		port       string
		want       bool
	}{
		// Empty hostname = match none
		{"empty matches none", "", "anything.com", "443", false},
		{"empty matches none port", "", "anything.com", "8443", false},

		// Single exact domain, default port 443
		{"exact domain port 443", "example.com", "example.com", "443", true},
		{"exact domain port 8443", "example.com", "example.com", "8443", false},
		{"exact domain wrong name", "example.com", "other.com", "443", false},

		// Domain with specific port
		{"specific port match", "example.com:8443", "example.com", "8443", true},
		{"specific port no match", "example.com:8443", "example.com", "443", false},

		// Domain with port 0 (all ports)
		{"all ports match 443", "example.com:0", "example.com", "443", true},
		{"all ports match 8443", "example.com:0", "example.com", "8443", true},
		{"all ports wrong domain", "example.com:0", "other.com", "443", false},

		// Wildcard domain
		{"wildcard subdomain", "*.example.com", "foo.example.com", "443", true},
		{"wildcard no match root", "*.example.com", "example.com", "443", false},
		{"wildcard deep match", "*.example.com", "a.b.example.com", "443", true},

		// Multiple entries
		{"multi first match", "example.com, *.test.com", "example.com", "443", true},
		{"multi second match", "example.com, *.test.com", "foo.test.com", "443", true},
		{"multi no match", "example.com, *.test.com", "other.com", "443", false},

		// Wildcard all
		{"wildcard all on 443", "*", "anything.com", "443", true},
		{"wildcard all on other port", "*", "anything.com", "8443", false},
		{"wildcard all ports", "*:0", "anything.com", "8443", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewHostnameFilter(tt.hostname)
			if err != nil {
				t.Fatalf("NewHostnameFilter(%q) error: %v", tt.hostname, err)
			}
			got := filter.Allow(tt.serverName, tt.port)
			if got != tt.want {
				t.Errorf("Allow(%q, %s) = %v, want %v", tt.serverName, tt.port, got, tt.want)
			}
		})
	}
}

func TestNilFilterAllow(t *testing.T) {
	var f *HostnameFilter
	if f.Allow("anything.com", "443") {
		t.Error("nil filter should not allow anything")
	}
}
