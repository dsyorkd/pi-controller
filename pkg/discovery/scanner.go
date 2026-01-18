// Package discovery provides node discovery services for Pi Controller
package discovery

import (
	"fmt"
	"net"
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
