package runtime

import (
	"context"
	"net/netip"
	"strings"
	"testing"
)

func TestNewMCPServiceRejectsUnsafeHTTPURLsBeforeConnecting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
	}{
		{"unsupported scheme", "file:///etc/passwd"},
		{"missing host", "https:///mcp"},
		{"credentials", "https://user:pass@example.com/mcp"},
		{"IPv4 loopback", "http://127.0.0.1/mcp"},
		{"IPv4 private", "http://10.0.0.1/mcp"},
		{"cloud metadata", "http://169.254.169.254/latest/meta-data"},
		{"IPv4 unspecified", "http://0.0.0.0/mcp"},
		{"IPv4 multicast", "http://224.0.0.1/mcp"},
		{"IPv4 reserved", "http://192.0.2.1/mcp"},
		{"IPv6 loopback", "http://[::1]/mcp"},
		{"IPv6 private", "http://[fd00::1]/mcp"},
		{"IPv6 link-local", "http://[fe80::1]/mcp"},
		{"IPv6 unspecified", "http://[::]/mcp"},
		{"IPv6 multicast", "http://[ff02::1]/mcp"},
		{"IPv6 reserved", "http://[2001:db8::1]/mcp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewMCPService(context.Background(), MCPServiceConfig{
				Name: "unsafe", Type: "streamable", URL: tt.url,
			})
			if err == nil || !strings.Contains(err.Error(), "unsafe MCP URL") {
				t.Fatalf("NewMCPService(%q) error = %v, want unsafe MCP URL", tt.url, err)
			}
		})
	}
}

func TestValidateMCPURLRejectsAnyUnsafeDNSAnswer(t *testing.T) {
	t.Parallel()

	lookup := func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{
			netip.MustParseAddr("93.184.216.34"),
			netip.MustParseAddr("127.0.0.1"),
		}, nil
	}

	if _, err := validateMCPURL(context.Background(), "https://mcp.example/mcp", false, lookup); err == nil {
		t.Fatal("validateMCPURL accepted a hostname with a loopback DNS answer")
	}
}

func TestValidateMCPURLAllowsPublicHTTPAndExplicitPrivateOptIn(t *testing.T) {
	t.Parallel()

	publicLookup := func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	}
	if _, err := validateMCPURL(context.Background(), "https://mcp.example/mcp", false, publicLookup); err != nil {
		t.Fatalf("public URL rejected: %v", err)
	}

	privateLookup := func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
	}
	if _, err := validateMCPURL(context.Background(), "http://localhost:8080/mcp", true, privateLookup); err != nil {
		t.Fatalf("opted-in local URL rejected: %v", err)
	}
}
