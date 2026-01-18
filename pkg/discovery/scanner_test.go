package discovery

import (
	"net"
	"testing"
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
