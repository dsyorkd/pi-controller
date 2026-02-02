package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiscoveryConfigDefaults(t *testing.T) {
	t.Run("production defaults", func(t *testing.T) {
		cfg := getProductionDefaults()

		// Discovery should be enabled by default
		assert.True(t, cfg.Discovery.Enabled)
		assert.Equal(t, "mdns", cfg.Discovery.Method)
		assert.Equal(t, 9091, cfg.Discovery.Port)
		assert.Equal(t, "30s", cfg.Discovery.Interval)
		assert.Equal(t, "5s", cfg.Discovery.Timeout)
		assert.Equal(t, "pi-controller", cfg.Discovery.ServiceName)
		assert.Equal(t, "_pi-controller._tcp", cfg.Discovery.ServiceType)

		// Network scanning defaults
		assert.Empty(t, cfg.Discovery.ScanRanges) // No default scan ranges
		assert.Equal(t, []int{9091}, cfg.Discovery.ScanPorts)
		assert.Equal(t, "2s", cfg.Discovery.ScanTimeout)
		assert.Equal(t, 10, cfg.Discovery.ScanConcurrency)
		assert.Equal(t, 100, cfg.Discovery.ScanRateLimit)
	})

	t.Run("development defaults", func(t *testing.T) {
		cfg := getDevelopmentDefaults()

		// Development should have same discovery settings as production
		assert.True(t, cfg.Discovery.Enabled)
		assert.Equal(t, "mdns", cfg.Discovery.Method)
		assert.Equal(t, 9091, cfg.Discovery.Port)
	})
}

func TestDiscoveryConfigFromFile(t *testing.T) {
	t.Run("should load discovery config from file", func(t *testing.T) {
		content := `
discovery:
  enabled: true
  method: "scan"
  interface: "eth0"
  port: 9092
  interval: "60s"
  timeout: "10s"
  service_name: "my-pi-controller"
  service_type: "_custom._tcp"
  static_nodes:
    - "192.168.1.10"
    - "192.168.1.20"
  scan_ranges:
    - "192.168.1.0/24"
    - "10.0.0.0/24"
  scan_ports:
    - 9091
    - 9092
    - 8080
  scan_timeout: "3s"
  scan_concurrency: 20
  scan_rate_limit: 200
`
		tmpfile, err := os.CreateTemp("", "discovery-config-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		cfg, err := Load(tmpfile.Name())
		assert.NoError(t, err)
		assert.True(t, cfg.Discovery.Enabled)
		assert.Equal(t, "scan", cfg.Discovery.Method)
		assert.Equal(t, "eth0", cfg.Discovery.Interface)
		assert.Equal(t, 9092, cfg.Discovery.Port)
		assert.Equal(t, "60s", cfg.Discovery.Interval)
		assert.Equal(t, "10s", cfg.Discovery.Timeout)
		assert.Equal(t, "my-pi-controller", cfg.Discovery.ServiceName)
		assert.Equal(t, "_custom._tcp", cfg.Discovery.ServiceType)
		assert.Equal(t, []string{"192.168.1.10", "192.168.1.20"}, cfg.Discovery.StaticNodes)
		assert.Equal(t, []string{"192.168.1.0/24", "10.0.0.0/24"}, cfg.Discovery.ScanRanges)
		assert.Equal(t, []int{9091, 9092, 8080}, cfg.Discovery.ScanPorts)
		assert.Equal(t, "3s", cfg.Discovery.ScanTimeout)
		assert.Equal(t, 20, cfg.Discovery.ScanConcurrency)
		assert.Equal(t, 200, cfg.Discovery.ScanRateLimit)
	})

	t.Run("should load minimal discovery config", func(t *testing.T) {
		content := `
discovery:
  enabled: false
`
		tmpfile, err := os.CreateTemp("", "discovery-minimal-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		cfg, err := Load(tmpfile.Name())
		assert.NoError(t, err)
		assert.False(t, cfg.Discovery.Enabled)
		// Other fields should have defaults
		assert.Equal(t, "mdns", cfg.Discovery.Method)
		assert.Equal(t, 9091, cfg.Discovery.Port)
	})

	t.Run("should handle single scan range", func(t *testing.T) {
		content := `
discovery:
  scan_ranges:
    - "192.168.1.0/24"
`
		tmpfile, err := os.CreateTemp("", "discovery-single-range-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		cfg, err := Load(tmpfile.Name())
		assert.NoError(t, err)
		assert.Len(t, cfg.Discovery.ScanRanges, 1)
		assert.Equal(t, "192.168.1.0/24", cfg.Discovery.ScanRanges[0])
	})

	t.Run("should handle multiple scan ranges in various CIDR formats", func(t *testing.T) {
		content := `
discovery:
  scan_ranges:
    - "192.168.1.0/24"
    - "10.0.0.0/16"
    - "172.16.0.0/12"
`
		tmpfile, err := os.CreateTemp("", "discovery-multi-range-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		cfg, err := Load(tmpfile.Name())
		assert.NoError(t, err)
		assert.Len(t, cfg.Discovery.ScanRanges, 3)
		assert.Equal(t, "192.168.1.0/24", cfg.Discovery.ScanRanges[0])
		assert.Equal(t, "10.0.0.0/16", cfg.Discovery.ScanRanges[1])
		assert.Equal(t, "172.16.0.0/12", cfg.Discovery.ScanRanges[2])
	})

	t.Run("should handle single port scan", func(t *testing.T) {
		content := `
discovery:
  scan_ports:
    - 9091
`
		tmpfile, err := os.CreateTemp("", "discovery-single-port-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		cfg, err := Load(tmpfile.Name())
		assert.NoError(t, err)
		assert.Len(t, cfg.Discovery.ScanPorts, 1)
		assert.Equal(t, 9091, cfg.Discovery.ScanPorts[0])
	})

	t.Run("should handle multiple port scans", func(t *testing.T) {
		content := `
discovery:
  scan_ports:
    - 22
    - 80
    - 443
    - 9091
`
		tmpfile, err := os.CreateTemp("", "discovery-multi-port-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		cfg, err := Load(tmpfile.Name())
		assert.NoError(t, err)
		assert.Len(t, cfg.Discovery.ScanPorts, 4)
		assert.Equal(t, 22, cfg.Discovery.ScanPorts[0])
		assert.Equal(t, 80, cfg.Discovery.ScanPorts[1])
		assert.Equal(t, 443, cfg.Discovery.ScanPorts[2])
		assert.Equal(t, 9091, cfg.Discovery.ScanPorts[3])
	})
}

func TestDiscoveryConfigMerging(t *testing.T) {
	t.Run("should merge file config with defaults", func(t *testing.T) {
		content := `
discovery:
  method: "scan"
  scan_ranges:
    - "192.168.1.0/24"
`
		tmpfile, err := os.CreateTemp("", "discovery-merge-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		cfg, err := Load(tmpfile.Name())
		assert.NoError(t, err)
		// File values
		assert.Equal(t, "scan", cfg.Discovery.Method)
		assert.Equal(t, []string{"192.168.1.0/24"}, cfg.Discovery.ScanRanges)
		// Default values should still apply
		assert.True(t, cfg.Discovery.Enabled)
		assert.Equal(t, 9091, cfg.Discovery.Port)
		assert.Equal(t, []int{9091}, cfg.Discovery.ScanPorts)
	})

	t.Run("should preserve all settings when loading", func(t *testing.T) {
		content := `
discovery:
  enabled: true
  method: "scan"
  interface: "eth0"
  port: 9092
  interval: "60s"
  timeout: "10s"
  service_name: "my-controller"
  service_type: "_custom._tcp"
  static_nodes:
    - "192.168.1.10:9091"
  scan_ranges:
    - "192.168.1.0/24"
  scan_ports:
    - 9091
    - 9092
  scan_timeout: "3s"
  scan_concurrency: 20
  scan_rate_limit: 200
`
		tmpfile, err := os.CreateTemp("", "discovery-full-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		cfg, err := Load(tmpfile.Name())
		assert.NoError(t, err)
		assert.True(t, cfg.Discovery.Enabled)
		assert.Equal(t, "scan", cfg.Discovery.Method)
		assert.Equal(t, "eth0", cfg.Discovery.Interface)
		assert.Equal(t, 9092, cfg.Discovery.Port)
		assert.Equal(t, "60s", cfg.Discovery.Interval)
		assert.Equal(t, "10s", cfg.Discovery.Timeout)
		assert.Equal(t, "my-controller", cfg.Discovery.ServiceName)
		assert.Equal(t, "_custom._tcp", cfg.Discovery.ServiceType)
		assert.Equal(t, []string{"192.168.1.10:9091"}, cfg.Discovery.StaticNodes)
		assert.Equal(t, []string{"192.168.1.0/24"}, cfg.Discovery.ScanRanges)
		assert.Equal(t, []int{9091, 9092}, cfg.Discovery.ScanPorts)
		assert.Equal(t, "3s", cfg.Discovery.ScanTimeout)
		assert.Equal(t, 20, cfg.Discovery.ScanConcurrency)
		assert.Equal(t, 200, cfg.Discovery.ScanRateLimit)
	})
}

func TestDiscoveryConfigValidation(t *testing.T) {
	t.Run("should accept valid mDNS configuration", func(t *testing.T) {
		content := `
discovery:
  enabled: true
  method: "mdns"
  service_name: "pi-controller"
  service_type: "_pi-controller._tcp"
`
		tmpfile, err := os.CreateTemp("", "discovery-valid-mdns-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		cfg, err := Load(tmpfile.Name())
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("should accept valid scan configuration", func(t *testing.T) {
		content := `
discovery:
  enabled: true
  method: "scan"
  scan_ranges:
    - "192.168.1.0/24"
  scan_ports:
    - 9091
  scan_timeout: "2s"
  scan_concurrency: 10
  scan_rate_limit: 100
`
		tmpfile, err := os.CreateTemp("", "discovery-valid-scan-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		cfg, err := Load(tmpfile.Name())
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("should accept valid static configuration", func(t *testing.T) {
		content := `
discovery:
  enabled: true
  method: "static"
  static_nodes:
    - "192.168.1.10:9091"
    - "192.168.1.20:9091"
`
		tmpfile, err := os.CreateTemp("", "discovery-valid-static-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		cfg, err := Load(tmpfile.Name())
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("should reject invalid YAML", func(t *testing.T) {
		content := `
discovery:
  enabled: true
  method: scan
  scan_ranges:
    - "192.168.1.0/24
invalid-yaml-here
`
		tmpfile, err := os.CreateTemp("", "discovery-invalid-yaml-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		_, err = Load(tmpfile.Name())
		assert.Error(t, err)
	})
}

func TestDiscoveryConfigMethods(t *testing.T) {
	t.Run("should support mdns method", func(t *testing.T) {
		cfg := DiscoveryConfig{
			Enabled: true,
			Method:  "mdns",
		}

		assert.Equal(t, "mdns", cfg.Method)
	})

	t.Run("should support scan method", func(t *testing.T) {
		cfg := DiscoveryConfig{
			Enabled: true,
			Method:  "scan",
		}

		assert.Equal(t, "scan", cfg.Method)
	})

	t.Run("should support static method", func(t *testing.T) {
		cfg := DiscoveryConfig{
			Enabled: true,
			Method:  "static",
		}

		assert.Equal(t, "static", cfg.Method)
	})
}

func TestDiscoveryConfigScanSettings(t *testing.T) {
	t.Run("should have reasonable scan concurrency", func(t *testing.T) {
		cfg := getProductionDefaults()

		assert.Equal(t, 10, cfg.Discovery.ScanConcurrency)
		assert.True(t, cfg.Discovery.ScanConcurrency > 0)
		assert.True(t, cfg.Discovery.ScanConcurrency <= 100)
	})

	t.Run("should have reasonable scan rate limit", func(t *testing.T) {
		cfg := getProductionDefaults()

		assert.Equal(t, 100, cfg.Discovery.ScanRateLimit)
		assert.True(t, cfg.Discovery.ScanRateLimit > 0)
	})

	t.Run("should have reasonable scan timeout", func(t *testing.T) {
		cfg := getProductionDefaults()

		assert.Equal(t, "2s", cfg.Discovery.ScanTimeout)
		assert.NotEmpty(t, cfg.Discovery.ScanTimeout)
	})

	t.Run("should allow custom scan settings", func(t *testing.T) {
		cfg := DiscoveryConfig{
			ScanConcurrency: 50,
			ScanRateLimit:   200,
			ScanTimeout:     "5s",
		}

		assert.Equal(t, 50, cfg.ScanConcurrency)
		assert.Equal(t, 200, cfg.ScanRateLimit)
		assert.Equal(t, "5s", cfg.ScanTimeout)
	})
}

func TestDiscoveryConfigStaticNodes(t *testing.T) {
	t.Run("should handle empty static nodes", func(t *testing.T) {
		cfg := DiscoveryConfig{
			StaticNodes: []string{},
		}

		assert.Empty(t, cfg.StaticNodes)
		assert.NotNil(t, cfg.StaticNodes)
	})

	t.Run("should handle single static node", func(t *testing.T) {
		cfg := DiscoveryConfig{
			StaticNodes: []string{"192.168.1.10:9091"},
		}

		assert.Len(t, cfg.StaticNodes, 1)
		assert.Equal(t, "192.168.1.10:9091", cfg.StaticNodes[0])
	})

	t.Run("should handle multiple static nodes", func(t *testing.T) {
		cfg := DiscoveryConfig{
			StaticNodes: []string{
				"192.168.1.10:9091",
				"192.168.1.20:9091",
				"192.168.1.30:9091",
			},
		}

		assert.Len(t, cfg.StaticNodes, 3)
		assert.Contains(t, cfg.StaticNodes, "192.168.1.10:9091")
		assert.Contains(t, cfg.StaticNodes, "192.168.1.20:9091")
		assert.Contains(t, cfg.StaticNodes, "192.168.1.30:9091")
	})
}

func TestDiscoveryConfigScanRanges(t *testing.T) {
	t.Run("should handle empty scan ranges", func(t *testing.T) {
		cfg := getProductionDefaults()

		assert.Empty(t, cfg.Discovery.ScanRanges)
		assert.NotNil(t, cfg.Discovery.ScanRanges)
	})

	t.Run("should handle single CIDR range", func(t *testing.T) {
		cfg := DiscoveryConfig{
			ScanRanges: []string{"192.168.1.0/24"},
		}

		assert.Len(t, cfg.ScanRanges, 1)
		assert.Equal(t, "192.168.1.0/24", cfg.ScanRanges[0])
	})

	t.Run("should handle multiple CIDR ranges", func(t *testing.T) {
		cfg := DiscoveryConfig{
			ScanRanges: []string{
				"192.168.1.0/24",
				"10.0.0.0/16",
				"172.16.0.0/12",
			},
		}

		assert.Len(t, cfg.ScanRanges, 3)
		assert.Contains(t, cfg.ScanRanges, "192.168.1.0/24")
		assert.Contains(t, cfg.ScanRanges, "10.0.0.0/16")
		assert.Contains(t, cfg.ScanRanges, "172.16.0.0/12")
	})

	t.Run("should handle /32 single host CIDR", func(t *testing.T) {
		cfg := DiscoveryConfig{
			ScanRanges: []string{"192.168.1.100/32"},
		}

		assert.Len(t, cfg.ScanRanges, 1)
		assert.Equal(t, "192.168.1.100/32", cfg.ScanRanges[0])
	})
}

func TestDiscoveryConfigScanPorts(t *testing.T) {
	t.Run("should have default scan port", func(t *testing.T) {
		cfg := getProductionDefaults()

		assert.Equal(t, []int{9091}, cfg.Discovery.ScanPorts)
		assert.Len(t, cfg.Discovery.ScanPorts, 1)
	})

	t.Run("should handle single custom port", func(t *testing.T) {
		cfg := DiscoveryConfig{
			ScanPorts: []int{8080},
		}

		assert.Len(t, cfg.ScanPorts, 1)
		assert.Equal(t, 8080, cfg.ScanPorts[0])
	})

	t.Run("should handle multiple ports", func(t *testing.T) {
		cfg := DiscoveryConfig{
			ScanPorts: []int{22, 80, 443, 9091, 9092},
		}

		assert.Len(t, cfg.ScanPorts, 5)
		assert.Contains(t, cfg.ScanPorts, 22)
		assert.Contains(t, cfg.ScanPorts, 80)
		assert.Contains(t, cfg.ScanPorts, 443)
		assert.Contains(t, cfg.ScanPorts, 9091)
		assert.Contains(t, cfg.ScanPorts, 9092)
	})
}
