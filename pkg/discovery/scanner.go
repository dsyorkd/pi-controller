// Package discovery provides node discovery services for Pi Controller
package discovery

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/time/rate"
)

// parseIPRange parses a CIDR notation string and returns all IP addresses in that range.
// Example: "192.168.1.0/24" returns all IPs from 192.168.1.0 to 192.168.1.255
func parseIPRange(cidr string) ([]net.IP, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR notation: %w", err)
	}

	var ips []net.IP
	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); incIP(ip) {
		// Create a copy of the IP to avoid all entries pointing to the same address
		ipCopy := make(net.IP, len(ip))
		copy(ipCopy, ip)
		ips = append(ips, ipCopy)
	}

	return ips, nil
}

// generateIPsFromRange generates all IP addresses between start and end (inclusive).
// Both start and end must be valid IPv4 or IPv6 addresses of the same type.
func generateIPsFromRange(start, end net.IP) []net.IP {
	// Normalize to 16-byte representation for consistent comparison
	start = start.To16()
	end = end.To16()

	if start == nil || end == nil {
		return nil
	}

	var ips []net.IP
	for ip := make(net.IP, len(start)); ; incIP(ip) {
		copy(ip, start)

		// Create a copy to avoid all entries pointing to the same address
		ipCopy := make(net.IP, len(ip))
		copy(ipCopy, ip)
		ips = append(ips, ipCopy)

		// Check if we've reached the end
		if ip.Equal(end) {
			break
		}

		// Advance start to next IP
		incIP(start)

		// Safety check to prevent infinite loops
		if compareIP(start, end) > 0 {
			break
		}
	}

	return ips
}

// incIP increments an IP address by 1
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// compareIP compares two IP addresses.
// Returns -1 if ip1 < ip2, 0 if ip1 == ip2, 1 if ip1 > ip2
func compareIP(ip1, ip2 net.IP) int {
	// Normalize to 16-byte representation
	ip1 = ip1.To16()
	ip2 = ip2.To16()

	if ip1 == nil || ip2 == nil {
		return 0
	}

	for i := 0; i < len(ip1); i++ {
		if ip1[i] < ip2[i] {
			return -1
		}
		if ip1[i] > ip2[i] {
			return 1
		}
	}
	return 0
}

// PortScanner handles TCP port scanning with rate limiting
type PortScanner struct {
	rateLimiter *rate.Limiter
	timeout     time.Duration
}

// NewPortScanner creates a new port scanner with the specified rate limit and timeout.
// rateLimit specifies the maximum number of scans per second.
// timeout specifies how long to wait for each connection attempt.
func NewPortScanner(rateLimit int, timeout time.Duration) *PortScanner {
	// Create a rate limiter with the specified rate and a burst of the same size
	// This allows for smooth rate limiting without strict bucket refills
	limiter := rate.NewLimiter(rate.Limit(rateLimit), rateLimit)

	return &PortScanner{
		rateLimiter: limiter,
		timeout:     timeout,
	}
}

// ScanPort attempts to connect to the specified IP and port within the timeout period.
// It respects the rate limiter to avoid network flooding.
// Returns true if the port is open (connection successful), false otherwise.
func (ps *PortScanner) ScanPort(ctx context.Context, ip net.IP, port int) bool {
	// Wait for rate limiter permission
	if err := ps.rateLimiter.Wait(ctx); err != nil {
		// Context was cancelled or deadline exceeded
		return false
	}

	return scanPort(ip, port, ps.timeout)
}

// scanPort performs a single TCP connection attempt to the specified IP and port.
// Returns true if the connection succeeds (port is open), false otherwise.
func scanPort(ip net.IP, port int, timeout time.Duration) bool {
	address := fmt.Sprintf("%s:%d", ip.String(), port)

	// Attempt to establish a TCP connection with the specified timeout
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		// Connection failed - port is closed or host is unreachable
		return false
	}

	// Connection succeeded - port is open
	// Close the connection immediately as we only need to verify it's open
	conn.Close()
	return true
}

// ScanResult represents the result of a port scan operation
type ScanResult struct {
	IP   net.IP
	Port int
	Open bool
}

// ScanPorts scans multiple ports on a single IP address concurrently.
// It respects the rate limiter and returns a slice of scan results.
func (ps *PortScanner) ScanPorts(ctx context.Context, ip net.IP, ports []int) []ScanResult {
	results := make([]ScanResult, len(ports))

	for i, port := range ports {
		// Check if context is cancelled before each scan
		select {
		case <-ctx.Done():
			// Context cancelled, return partial results
			return results[:i]
		default:
		}

		isOpen := ps.ScanPort(ctx, ip, port)
		results[i] = ScanResult{
			IP:   ip,
			Port: port,
			Open: isOpen,
		}
	}

	return results
}

// ScanRange scans a range of IP addresses on the specified ports.
// It returns a channel of ScanResults for ports that are open.
// The scan respects the rate limiter and can be cancelled via the context.
func (ps *PortScanner) ScanRange(ctx context.Context, ips []net.IP, ports []int) <-chan ScanResult {
	results := make(chan ScanResult, 100) // Buffered channel to avoid blocking

	go func() {
		defer close(results)

		for _, ip := range ips {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return
			default:
			}

			for _, port := range ports {
				// Check if context is cancelled before each scan
				select {
				case <-ctx.Done():
					return
				default:
				}

				isOpen := ps.ScanPort(ctx, ip, port)
				if isOpen {
					// Only send results for open ports to reduce noise
					results <- ScanResult{
						IP:   ip,
						Port: port,
						Open: true,
					}
				}
			}
		}
	}()

	return results
}
