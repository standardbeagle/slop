package runtime

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewMCPCommandUsesExplicitEnvironmentOnly(t *testing.T) {
	cmd := newMCPCommand(MCPServiceConfig{Command: "go"})
	if cmd.Env == nil {
		t.Fatal("command environment is nil, which inherits the host environment")
	}
	if len(cmd.Env) != 0 {
		t.Fatalf("default command environment = %v, want empty", cmd.Env)
	}
	if cmd.Path == "" || cmd.Path == "go" {
		t.Fatalf("command path = %q, want executable resolved from host PATH", cmd.Path)
	}

	cmd = newMCPCommand(MCPServiceConfig{
		Command: "go",
		Env:     []string{"MCP_ALLOWED=value"},
	})
	if got, want := cmd.Env, []string{"MCP_ALLOWED=value"}; len(got) != 1 || got[0] != want[0] {
		t.Fatalf("explicit command environment = %v, want %v", got, want)
	}
}

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

func TestMetadataEndpointsRemainBlockedWithPrivateOptIn(t *testing.T) {
	t.Parallel()

	metadata := []netip.Addr{
		netip.MustParseAddr("169.254.169.254"),
		netip.MustParseAddr("::ffff:169.254.169.254"),
		netip.MustParseAddr("fd00:ec2::254"),
	}
	for _, addr := range metadata {
		addr := addr
		t.Run(addr.String(), func(t *testing.T) {
			t.Parallel()
			literalURL := "http://" + addr.String() + "/mcp"
			if addr.Is6() {
				literalURL = "http://[" + addr.String() + "]/mcp"
			}
			if _, err := validateMCPURL(context.Background(), literalURL, true, nil); err == nil {
				t.Fatal("metadata literal accepted with private opt-in")
			}

			lookup := func(context.Context, string) ([]netip.Addr, error) { return []netip.Addr{addr}, nil }
			if _, err := validateMCPURL(context.Background(), "http://metadata.internal/mcp", true, lookup); err == nil {
				t.Fatal("metadata DNS answer accepted with private opt-in")
			}

			publicLookup := func(context.Context, string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
			}
			client, err := newMCPHTTPClientWithLookup(context.Background(), "https://mcp.example/mcp", true, publicLookup)
			if err != nil {
				t.Fatal(err)
			}
			req, err := http.NewRequest(http.MethodGet, literalURL, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := client.CheckRedirect(req, nil); err == nil {
				t.Fatal("metadata redirect accepted with private opt-in")
			}
		})
	}
}

func TestSpecialPurposeAddressesAreBlocked(t *testing.T) {
	t.Parallel()

	addresses := []string{
		"0.1.2.3", "100.64.0.1", "192.0.0.9", "192.31.196.1", "192.52.193.1",
		"192.88.99.1", "192.175.48.1", "198.18.0.1", "240.0.0.1",
		"64:ff9b::1", "64:ff9b:1::1", "100::1", "2001::1", "2002::1", "2620:4f:8000::1",
		"3fff::1", "5f00::1", "fec0::1",
	}
	for _, raw := range addresses {
		addr := netip.MustParseAddr(raw)
		if !unsafeMCPAddr(addr) {
			t.Errorf("unsafeMCPAddr(%s) = false, want true", addr)
		}
	}
}

func TestMCPDialFallsBackAcrossValidatedAddresses(t *testing.T) {
	t.Parallel()

	first := netip.MustParseAddr("93.184.216.34")
	second := netip.MustParseAddr("93.184.216.35")
	lookup := func(context.Context, string) ([]netip.Addr, error) { return []netip.Addr{first, second}, nil }
	var attempted []string
	var attemptedMu sync.Mutex
	dial := func(_ context.Context, _, address string) (net.Conn, error) {
		attemptedMu.Lock()
		defer attemptedMu.Unlock()
		attempted = append(attempted, address)
		if len(attempted) == 1 {
			return nil, errors.New("first address unavailable")
		}
		client, server := net.Pipe()
		server.Close()
		return client, nil
	}
	client, err := newMCPHTTPClientWithNetwork(context.Background(), "http://mcp.example/mcp", false, lookup, dial)
	if err != nil {
		t.Fatal(err)
	}
	transport := client.Transport.(*http.Transport)
	conn, err := transport.DialContext(context.Background(), "tcp", "mcp.example:80")
	if err != nil {
		t.Fatalf("DialContext did not fall back: %v", err)
	}
	conn.Close()
	want := []string{first.String() + ":80", second.String() + ":80"}
	if len(attempted) != 2 || attempted[0] != want[0] || attempted[1] != want[1] {
		t.Fatalf("attempted = %v, want %v", attempted, want)
	}
}

func TestMCPDialStaggersCandidatesWithoutReresolving(t *testing.T) {
	t.Parallel()

	addrs := []netip.Addr{
		netip.MustParseAddr("93.184.216.34"),
		netip.MustParseAddr("93.184.216.35"),
	}
	var lookups atomic.Int32
	lookup := func(context.Context, string) ([]netip.Addr, error) {
		lookups.Add(1)
		return addrs, nil
	}
	dial := func(ctx context.Context, _, address string) (net.Conn, error) {
		if address == addrs[0].String()+":80" {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		client, server := net.Pipe()
		server.Close()
		return client, nil
	}
	client, err := newMCPHTTPClientWithNetworkDelay(
		context.Background(), "http://mcp.example/mcp", false, lookup, dial, 5*time.Millisecond,
	)
	if err != nil {
		t.Fatal(err)
	}
	lookups.Store(0) // Ignore the constructor's pre-connect validation lookup.

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	transport := client.Transport.(*http.Transport)
	conn, err := transport.DialContext(ctx, "tcp", "mcp.example:80")
	if err != nil {
		t.Fatalf("staggered dial failed: %v", err)
	}
	conn.Close()
	if got := lookups.Load(); got != 1 {
		t.Fatalf("dial performed %d DNS lookups, want one validated snapshot", got)
	}
}

func TestMCPDialClosesLateLosingConnection(t *testing.T) {
	t.Parallel()

	addrs := []netip.Addr{netip.MustParseAddr("93.184.216.34"), netip.MustParseAddr("93.184.216.35")}
	lookup := func(context.Context, string) ([]netip.Addr, error) { return addrs, nil }
	loserClosed := make(chan struct{})
	dial := func(ctx context.Context, _, address string) (net.Conn, error) {
		client, server := net.Pipe()
		server.Close()
		if address == addrs[0].String()+":80" {
			<-ctx.Done()
			return &closeSignalConn{Conn: client, closed: loserClosed}, nil
		}
		return client, nil
	}
	client, err := newMCPHTTPClientWithNetworkDelay(
		context.Background(), "http://mcp.example/mcp", false, lookup, dial, time.Millisecond,
	)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := client.Transport.(*http.Transport).DialContext(context.Background(), "tcp", "mcp.example:80")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	select {
	case <-loserClosed:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("late losing connection was not closed")
	}
}

func TestMCPDialAttemptBudgetSchedulesLargeAddressSetAsCapacityReturns(t *testing.T) {
	addrs := make([]netip.Addr, 64)
	for i := range addrs {
		addrs[i] = netip.AddrFrom4([4]byte{93, 184, 216, byte(i + 1)})
	}

	started := make(chan string, len(addrs))
	release := make(chan struct{}, len(addrs))
	var active atomic.Int32
	var maximum atomic.Int32
	dial := func(_ context.Context, _, address string) (net.Conn, error) {
		current := active.Add(1)
		for old := maximum.Load(); current > old && !maximum.CompareAndSwap(old, current); old = maximum.Load() {
		}
		started <- address
		<-release
		active.Add(-1)
		return nil, errors.New("unavailable")
	}

	done := make(chan error, 1)
	go func() {
		_, err := dialMCPAddresses(context.Background(), "tcp", "80", addrs, dial, 0)
		done <- err
	}()

	for range mcpDialAttemptBudget {
		<-started
	}
	select {
	case address := <-started:
		t.Fatalf("dial started beyond attempt budget: %s", address)
	default:
	}
	if got := maximum.Load(); got != mcpDialAttemptBudget {
		t.Fatalf("maximum live attempts = %d, want %d", got, mcpDialAttemptBudget)
	}

	release <- struct{}{}
	if got, want := <-started, net.JoinHostPort(addrs[mcpDialAttemptBudget].String(), "80"); got != want {
		t.Fatalf("next scheduled address = %q, want %q", got, want)
	}

	for i := 0; i < len(addrs)-mcpDialAttemptBudget-1; i++ {
		release <- struct{}{}
		<-started
	}
	for range mcpDialAttemptBudget {
		release <- struct{}{}
	}
	if err := <-done; err == nil {
		t.Fatal("dial succeeded, want all-attempts error")
	}
	if got := active.Load(); got != 0 {
		t.Fatalf("live attempts after return = %d, want zero", got)
	}
}

func TestMCPDialCancellationJoinsAttemptWorkers(t *testing.T) {
	addrs := make([]netip.Addr, 32)
	for i := range addrs {
		addrs[i] = netip.AddrFrom4([4]byte{93, 184, 216, byte(i + 1)})
	}

	started := make(chan struct{}, mcpDialAttemptBudget)
	exited := make(chan struct{}, mcpDialAttemptBudget)
	dial := func(ctx context.Context, _, _ string) (net.Conn, error) {
		started <- struct{}{}
		<-ctx.Done()
		exited <- struct{}{}
		return nil, ctx.Err()
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := dialMCPAddresses(ctx, "tcp", "80", addrs, dial, 0)
		done <- err
	}()
	for range mcpDialAttemptBudget {
		<-started
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("dial error = %v, want context.Canceled", err)
	}
	for range mcpDialAttemptBudget {
		<-exited
	}
}

type closeSignalConn struct {
	net.Conn
	closed chan struct{}
	once   sync.Once
}

func (c *closeSignalConn) Close() error {
	c.once.Do(func() { close(c.closed) })
	return c.Conn.Close()
}

func TestMCPHTTPClientRejectsUnsafeRedirect(t *testing.T) {
	t.Parallel()

	lookup := func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	}
	client, err := newMCPHTTPClientWithLookup(context.Background(), "https://mcp.example/mcp", false, lookup)
	if err != nil {
		t.Fatalf("newMCPHTTPClientWithLookup: %v", err)
	}
	req, err := http.NewRequest(http.MethodGet, "http://169.254.169.254/latest/meta-data", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.CheckRedirect(req, nil); err == nil {
		t.Fatal("redirect to cloud metadata endpoint was accepted")
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
	if _, err := validateMCPURL(context.Background(), "http://169.254.169.254/mcp", true, privateLookup); err == nil {
		t.Fatal("private-network opt-in accepted a cloud metadata endpoint")
	}
}
