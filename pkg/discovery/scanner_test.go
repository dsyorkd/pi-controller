package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestParseIPRange(t *testing.T) {
	tests := []struct {
		name        string
		cidr        string
		wantCount   int
		wantFirst   string
		wantLast    string
		expectError bool
	}{
		{
			name:        "valid /30 network",
			cidr:        "192.168.1.0/30",
			wantCount:   4,
			wantFirst:   "192.168.1.0",
			wantLast:    "192.168.1.3",
			expectError: false,
		},
		{
			name:        "valid /29 network",
			cidr:        "10.0.0.0/29",
			wantCount:   8,
			wantFirst:   "10.0.0.0",
			wantLast:    "10.0.0.7",
			expectError: false,
		},
		{
			name:        "valid /24 network",
			cidr:        "192.168.1.0/24",
			wantCount:   256,
			wantFirst:   "192.168.1.0",
			wantLast:    "192.168.1.255",
			expectError: false,
		},
		{
			name:        "single host /32",
			cidr:        "192.168.1.100/32",
			wantCount:   1,
			wantFirst:   "192.168.1.100",
			wantLast:    "192.168.1.100",
			expectError: false,
		},
		{
			name:        "invalid CIDR format",
			cidr:        "not-a-cidr",
			expectError: true,
		},
		{
			name:        "invalid IP address",
			cidr:        "999.999.999.999/24",
			expectError: true,
		},
		{
			name:        "missing mask",
			cidr:        "192.168.1.0",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ips, err := parseIPRange(tt.cidr)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(ips) != tt.wantCount {
				t.Errorf("got %d IPs, want %d", len(ips), tt.wantCount)
			}

			if len(ips) > 0 {
				if ips[0].String() != tt.wantFirst {
					t.Errorf("first IP = %s, want %s", ips[0].String(), tt.wantFirst)
				}
				if ips[len(ips)-1].String() != tt.wantLast {
					t.Errorf("last IP = %s, want %s", ips[len(ips)-1].String(), tt.wantLast)
				}
			}
		})
	}
}

func TestGenerateIPRange(t *testing.T) {
	tests := []struct {
		name      string
		start     string
		end       string
		wantCount int
		wantFirst string
		wantLast  string
	}{
		{
			name:      "small range",
			start:     "192.168.1.1",
			end:       "192.168.1.5",
			wantCount: 5,
			wantFirst: "192.168.1.1",
			wantLast:  "192.168.1.5",
		},
		{
			name:      "single IP",
			start:     "10.0.0.1",
			end:       "10.0.0.1",
			wantCount: 1,
			wantFirst: "10.0.0.1",
			wantLast:  "10.0.0.1",
		},
		{
			name:      "cross subnet boundary",
			start:     "192.168.1.254",
			end:       "192.168.2.2",
			wantCount: 5,
			wantFirst: "192.168.1.254",
			wantLast:  "192.168.2.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := net.ParseIP(tt.start)
			end := net.ParseIP(tt.end)

			if start == nil || end == nil {
				t.Fatalf("failed to parse test IPs: start=%s, end=%s", tt.start, tt.end)
			}

			ips := generateIPsFromRange(start, end)

			if len(ips) != tt.wantCount {
				t.Errorf("got %d IPs, want %d", len(ips), tt.wantCount)
			}

			if len(ips) > 0 {
				// Compare using To4() for IPv4 addresses to handle representation differences
				firstStr := ips[0].String()
				lastStr := ips[len(ips)-1].String()

				if firstStr != tt.wantFirst {
					t.Errorf("first IP = %s, want %s", firstStr, tt.wantFirst)
				}
				if lastStr != tt.wantLast {
					t.Errorf("last IP = %s, want %s", lastStr, tt.wantLast)
				}
			}
		})
	}
}

func TestGenerateIPRangeInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		start string
		end   string
	}{
		{
			name:  "nil start IP",
			start: "invalid",
			end:   "192.168.1.10",
		},
		{
			name:  "nil end IP",
			start: "192.168.1.1",
			end:   "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := net.ParseIP(tt.start)
			end := net.ParseIP(tt.end)

			ips := generateIPsFromRange(start, end)
			if ips != nil {
				t.Errorf("expected nil result for invalid input, got %d IPs", len(ips))
			}
		})
	}
}

func TestIncIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want string
	}{
		{
			name: "simple increment",
			ip:   "192.168.1.1",
			want: "192.168.1.2",
		},
		{
			name: "roll over octet",
			ip:   "192.168.1.255",
			want: "192.168.2.0",
		},
		{
			name: "roll over multiple octets",
			ip:   "192.168.255.255",
			want: "192.169.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse test IP: %s", tt.ip)
			}

			incIP(ip)

			if ip.String() != tt.want {
				t.Errorf("incIP() = %s, want %s", ip.String(), tt.want)
			}
		})
	}
}

func TestCompareIP(t *testing.T) {
	tests := []struct {
		name string
		ip1  string
		ip2  string
		want int
	}{
		{
			name: "equal IPs",
			ip1:  "192.168.1.1",
			ip2:  "192.168.1.1",
			want: 0,
		},
		{
			name: "ip1 less than ip2",
			ip1:  "192.168.1.1",
			ip2:  "192.168.1.2",
			want: -1,
		},
		{
			name: "ip1 greater than ip2",
			ip1:  "192.168.1.10",
			ip2:  "192.168.1.5",
			want: 1,
		},
		{
			name: "different subnets",
			ip1:  "192.168.2.1",
			ip2:  "192.168.1.1",
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip1 := net.ParseIP(tt.ip1)
			ip2 := net.ParseIP(tt.ip2)

			if ip1 == nil || ip2 == nil {
				t.Fatalf("failed to parse test IPs: ip1=%s, ip2=%s", tt.ip1, tt.ip2)
			}

			got := compareIP(ip1, ip2)
			if got != tt.want {
				t.Errorf("compareIP(%s, %s) = %d, want %d", tt.ip1, tt.ip2, got, tt.want)
			}
		})
	}
}

func TestNewPortScanner(t *testing.T) {
	rateLimit := 100
	timeout := 2 * time.Second

	scanner := NewPortScanner(rateLimit, timeout)

	if scanner == nil {
		t.Fatal("NewPortScanner returned nil")
	}

	if scanner.rateLimiter == nil {
		t.Error("rateLimiter is nil")
	}

	if scanner.timeout != timeout {
		t.Errorf("timeout = %v, want %v", scanner.timeout, timeout)
	}
}

func TestScanPort(t *testing.T) {
	// Start a test TCP server on a random available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer listener.Close()

	// Get the actual port the server is listening on
	addr := listener.Addr().(*net.TCPAddr)
	openPort := addr.Port

	// Accept connections in the background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	tests := []struct {
		name    string
		ip      string
		port    int
		timeout time.Duration
		want    bool
	}{
		{
			name:    "open port",
			ip:      "127.0.0.1",
			port:    openPort,
			timeout: 1 * time.Second,
			want:    true,
		},
		{
			name:    "closed port",
			ip:      "127.0.0.1",
			port:    65534, // Very unlikely to be open
			timeout: 100 * time.Millisecond,
			want:    false,
		},
		{
			name:    "unreachable host",
			ip:      "192.0.2.1", // TEST-NET-1, should be unreachable
			port:    80,
			timeout: 100 * time.Millisecond,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP: %s", tt.ip)
			}

			got := scanPort(ip, tt.port, tt.timeout)
			if got != tt.want {
				t.Errorf("scanPort(%s, %d, %v) = %v, want %v", tt.ip, tt.port, tt.timeout, got, tt.want)
			}
		})
	}
}

func TestPortScannerScanPort(t *testing.T) {
	// Start a test TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	openPort := addr.Port

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	scanner := NewPortScanner(100, 1*time.Second)
	ctx := context.Background()

	ip := net.ParseIP("127.0.0.1")
	if ip == nil {
		t.Fatal("failed to parse IP")
	}

	// Test scanning an open port
	if !scanner.ScanPort(ctx, ip, openPort) {
		t.Error("expected port to be open")
	}

	// Test scanning a closed port
	if scanner.ScanPort(ctx, ip, 65534) {
		t.Error("expected port to be closed")
	}
}

func TestPortScannerRateLimiting(t *testing.T) {
	// Create a scanner with a low rate limit
	rateLimit := 10 // 10 scans per second
	scanner := NewPortScanner(rateLimit, 100*time.Millisecond)
	ctx := context.Background()

	ip := net.ParseIP("127.0.0.1")
	closedPort := 65534

	// Perform multiple scans and measure the time taken
	numScans := 20
	start := time.Now()

	for i := 0; i < numScans; i++ {
		scanner.ScanPort(ctx, ip, closedPort)
	}

	elapsed := time.Since(start)

	// With rate limiting of 10 scans/second, 20 scans should take at least 2 seconds
	// We'll be lenient and check for at least 1.5 seconds to account for timing variations
	minExpected := time.Duration(float64(numScans)/float64(rateLimit)*0.75) * time.Second
	if elapsed < minExpected {
		t.Errorf("rate limiting not working: %d scans took %v, expected at least %v", numScans, elapsed, minExpected)
	}
}

func TestPortScannerContextCancellation(t *testing.T) {
	scanner := NewPortScanner(10, 1*time.Second)

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	ip := net.ParseIP("127.0.0.1")

	// ScanPort should return false when context is cancelled
	result := scanner.ScanPort(ctx, ip, 80)
	if result {
		t.Error("expected ScanPort to return false when context is cancelled")
	}
}

func TestScanPorts(t *testing.T) {
	// Start two test TCP servers
	listener1, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server 1: %v", err)
	}
	defer listener1.Close()

	listener2, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server 2: %v", err)
	}
	defer listener2.Close()

	port1 := listener1.Addr().(*net.TCPAddr).Port
	port2 := listener2.Addr().(*net.TCPAddr).Port

	// Accept connections
	go func() {
		for {
			conn, err := listener1.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	go func() {
		for {
			conn, err := listener2.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	scanner := NewPortScanner(100, 1*time.Second)
	ctx := context.Background()

	ip := net.ParseIP("127.0.0.1")
	ports := []int{port1, port2, 65534} // Two open, one closed

	results := scanner.ScanPorts(ctx, ip, ports)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Check that the two open ports are detected
	openCount := 0
	for _, result := range results {
		if result.Open {
			openCount++
			if result.Port != port1 && result.Port != port2 {
				t.Errorf("unexpected open port: %d", result.Port)
			}
		}
	}

	if openCount != 2 {
		t.Errorf("expected 2 open ports, got %d", openCount)
	}
}

func TestScanRange(t *testing.T) {
	// Start a test TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer listener.Close()

	openPort := listener.Addr().(*net.TCPAddr).Port

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	scanner := NewPortScanner(100, 500*time.Millisecond)
	ctx := context.Background()

	// Scan a small range including localhost
	ips := []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("127.0.0.2"), // Unlikely to have anything listening
	}
	ports := []int{openPort, 65534}

	resultChan := scanner.ScanRange(ctx, ips, ports)

	var results []ScanResult
	for result := range resultChan {
		results = append(results, result)
	}

	// We should find at least one open port (our test server)
	if len(results) == 0 {
		t.Error("expected at least one open port, got none")
	}

	// Verify the result is for our test server
	found := false
	for _, result := range results {
		if result.IP.String() == "127.0.0.1" && result.Port == openPort && result.Open {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("did not find expected open port %d on 127.0.0.1", openPort)
	}
}

func TestScanRangeContextCancellation(t *testing.T) {
	scanner := NewPortScanner(10, 1*time.Second)

	// Create a large IP range that would take a while to scan
	ips := make([]net.IP, 100)
	for i := 0; i < 100; i++ {
		ips[i] = net.ParseIP(fmt.Sprintf("192.0.2.%d", i))
	}
	ports := []int{80, 443, 8080}

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	resultChan := scanner.ScanRange(ctx, ips, ports)

	// Collect results until channel closes
	var results []ScanResult
	for result := range resultChan {
		results = append(results, result)
	}

	// With the timeout, we should not scan all 300 combinations (100 IPs * 3 ports)
	// We're scanning unreachable IPs so we shouldn't get any open ports anyway
	t.Logf("Scanned %d combinations before context cancellation", len(results))
}

func TestIdentifyNode(t *testing.T) {
	// Start a test HTTP server that mimics the pi-controller health endpoint
	handler := http.NewServeMux()
	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   "v1.0.0",
			"uptime":    "1h30m",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	server := &http.Server{
		Handler: handler,
	}

	// Start server on a random available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port

	// Start the server in a goroutine
	go func() {
		server.Serve(listener)
	}()
	defer server.Close()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name        string
		ip          string
		port        int
		timeout     time.Duration
		expectError bool
		checkNode   func(*testing.T, *Node)
	}{
		{
			name:        "successful identification",
			ip:          "127.0.0.1",
			port:        port,
			timeout:     2 * time.Second,
			expectError: false,
			checkNode: func(t *testing.T, node *Node) {
				if node == nil {
					t.Fatal("expected non-nil node")
				}
				expectedID := fmt.Sprintf("scan-127.0.0.1-%d", port)
				if node.ID != expectedID {
					t.Errorf("ID = %s, want %s", node.ID, expectedID)
				}
				if node.Name != "127.0.0.1" {
					t.Errorf("Name = %s, want 127.0.0.1", node.Name)
				}
				if node.IPAddress != "127.0.0.1" {
					t.Errorf("IPAddress = %s, want 127.0.0.1", node.IPAddress)
				}
				if node.Port != port {
					t.Errorf("Port = %d, want %d", node.Port, port)
				}
				if node.ServiceType != "network_scan" {
					t.Errorf("ServiceType = %s, want network_scan", node.ServiceType)
				}
				if node.TXTRecords["version"] != "v1.0.0" {
					t.Errorf("TXTRecords[version] = %s, want v1.0.0", node.TXTRecords["version"])
				}
				if node.TXTRecords["uptime"] != "1h30m" {
					t.Errorf("TXTRecords[uptime] = %s, want 1h30m", node.TXTRecords["uptime"])
				}
			},
		},
		{
			name:        "unreachable host",
			ip:          "192.0.2.1", // TEST-NET-1, should be unreachable
			port:        80,
			timeout:     100 * time.Millisecond,
			expectError: true,
		},
		{
			name:        "wrong port",
			ip:          "127.0.0.1",
			port:        65534, // Unlikely to be open
			timeout:     100 * time.Millisecond,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP: %s", tt.ip)
			}

			node, err := identifyNode(ip, tt.port, tt.timeout)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if node != nil {
					t.Error("expected nil node on error")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.checkNode != nil {
				tt.checkNode(t, node)
			}
		})
	}
}

func TestIdentifyNodeInvalidResponse(t *testing.T) {
	tests := []struct {
		name        string
		handlerFunc http.HandlerFunc
		description string
	}{
		{
			name: "non-200 status code",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			description: "should fail when health endpoint returns non-200 status",
		},
		{
			name: "invalid JSON",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("not json"))
			},
			description: "should fail when health endpoint returns invalid JSON",
		},
		{
			name: "non-ok status",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				response := map[string]interface{}{
					"status":    "error",
					"timestamp": time.Now().Format(time.RFC3339),
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			},
			description: "should fail when health status is not 'ok'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start a test HTTP server with the custom handler
			handler := http.NewServeMux()
			handler.HandleFunc("/health", tt.handlerFunc)

			server := &http.Server{
				Handler: handler,
			}

			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("failed to start test server: %v", err)
			}
			defer listener.Close()

			addr := listener.Addr().(*net.TCPAddr)
			port := addr.Port

			go func() {
				server.Serve(listener)
			}()
			defer server.Close()

			// Give the server a moment to start
			time.Sleep(100 * time.Millisecond)

			ip := net.ParseIP("127.0.0.1")
			node, err := identifyNode(ip, port, 2*time.Second)

			if err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
			}
			if node != nil {
				t.Errorf("%s: expected nil node on error", tt.description)
			}
		})
	}
}
