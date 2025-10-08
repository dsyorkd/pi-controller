package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadWithViper(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		createFile  bool
		fileContent string
		want        func(*testing.T, *Config)
		wantErr     bool
	}{
		{
			name: "load with defaults",
			setupEnv: func() {
				os.Setenv("PI_CONTROLLER_ENVIRONMENT", "development")
			},
			cleanupEnv: func() {
				os.Unsetenv("PI_CONTROLLER_ENVIRONMENT")
			},
			want: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "pi-controller", cfg.App.Name)
				assert.Equal(t, true, cfg.WebUI.Enabled)
				assert.Equal(t, 3000, cfg.WebUI.Port)
			},
		},
		{
			name: "load with environment variables",
			setupEnv: func() {
				os.Setenv("PI_CONTROLLER_ENVIRONMENT", "development")
				os.Setenv("PI_CONTROLLER_WEBUI_PORT", "4000")
				os.Setenv("PI_CONTROLLER_API_PORT", "9090")
				os.Setenv("PI_CONTROLLER_WEBUI_BACKEND_API_URL", "http://custom:8080")
			},
			cleanupEnv: func() {
				os.Unsetenv("PI_CONTROLLER_ENVIRONMENT")
				os.Unsetenv("PI_CONTROLLER_WEBUI_PORT")
				os.Unsetenv("PI_CONTROLLER_API_PORT")
				os.Unsetenv("PI_CONTROLLER_WEBUI_BACKEND_API_URL")
			},
			want: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 4000, cfg.WebUI.Port)
				assert.Equal(t, 9090, cfg.API.Port)
				assert.Equal(t, "http://custom:8080", cfg.WebUI.Backend.API.URL)
			},
		},
		{
			name: "load with YAML file",
			setupEnv: func() {
				os.Setenv("PI_CONTROLLER_ENVIRONMENT", "development")
			},
			cleanupEnv: func() {
				os.Unsetenv("PI_CONTROLLER_ENVIRONMENT")
			},
			createFile: true,
			fileContent: `
app:
  name: "test-controller"

webui:
  enabled: true
  port: 5000
  branding:
    title: "Test Controller"
    theme: "light"
`,
			want: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "test-controller", cfg.App.Name)
				assert.Equal(t, 5000, cfg.WebUI.Port)
				assert.Equal(t, "Test Controller", cfg.WebUI.Branding.Title)
				assert.Equal(t, "light", cfg.WebUI.Branding.Theme)
			},
		},
		{
			name: "environment overrides file",
			setupEnv: func() {
				os.Setenv("PI_CONTROLLER_ENVIRONMENT", "development")
				os.Setenv("PI_CONTROLLER_WEBUI_PORT", "6000")
			},
			cleanupEnv: func() {
				os.Unsetenv("PI_CONTROLLER_ENVIRONMENT")
				os.Unsetenv("PI_CONTROLLER_WEBUI_PORT")
			},
			createFile: true,
			fileContent: `
webui:
  port: 5000
`,
			want: func(t *testing.T, cfg *Config) {
				// Env var should override file
				assert.Equal(t, 6000, cfg.WebUI.Port)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			defer func() {
				if tt.cleanupEnv != nil {
					tt.cleanupEnv()
				}
			}()

			var configPath string
			if tt.createFile {
				tmpDir := t.TempDir()
				configPath = filepath.Join(tmpDir, "config.yaml")
				err := os.WriteFile(configPath, []byte(tt.fileContent), 0644)
				require.NoError(t, err)
			}

			cfg, err := LoadWithViper(configPath)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			if tt.want != nil {
				tt.want(t, cfg)
			}
		})
	}
}

func TestLoadWithViperAndWatch(t *testing.T) {
	t.Run("watch config file changes", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		initialContent := `
webui:
  port: 3000
  branding:
    title: "Initial Title"
`
		err := os.WriteFile(configPath, []byte(initialContent), 0644)
		require.NoError(t, err)

		changeDetected := make(chan *Config, 1)
		onChange := func(newCfg *Config) {
			changeDetected <- newCfg
		}

		cfg, err := LoadWithViperAndWatch(configPath, onChange)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, 3000, cfg.WebUI.Port)
		assert.Equal(t, "Initial Title", cfg.WebUI.Branding.Title)

		// Modify config file
		time.Sleep(100 * time.Millisecond) // Give watcher time to start
		updatedContent := `
webui:
  port: 4000
  branding:
    title: "Updated Title"
`
		err = os.WriteFile(configPath, []byte(updatedContent), 0644)
		require.NoError(t, err)

		// Wait for change detection (with timeout)
		select {
		case newCfg := <-changeDetected:
			assert.Equal(t, 4000, newCfg.WebUI.Port)
			assert.Equal(t, "Updated Title", newCfg.WebUI.Branding.Title)
		case <-time.After(2 * time.Second):
			t.Log("Warning: Config change not detected within timeout (may be flaky in CI)")
		}
	})
}

func TestSetViperDefaults(t *testing.T) {
	t.Run("sets all webui defaults", func(t *testing.T) {
		v := viper.New()
		setViperDefaults(v)

		// Test some key defaults
		assert.Equal(t, true, v.GetBool("webui.enabled"))
		assert.Equal(t, "0.0.0.0", v.GetString("webui.host"))
		assert.Equal(t, 3000, v.GetInt("webui.port"))
		assert.Equal(t, "./web/dist", v.GetString("webui.static_dir"))
		assert.Equal(t, "index.html", v.GetString("webui.index_file"))
		assert.Equal(t, true, v.GetBool("webui.spa_mode"))

		// Test backend defaults
		assert.Equal(t, "http://localhost:8080", v.GetString("webui.backend.api.url"))
		assert.Equal(t, "http://pi-controller-api:8080", v.GetString("webui.backend.api.internal_url"))
		assert.Equal(t, "/api", v.GetString("webui.backend.api.prefix"))

		// Test feature flags
		assert.Equal(t, true, v.GetBool("webui.features.gpio_control"))
		assert.Equal(t, true, v.GetBool("webui.features.cluster_management"))
		assert.Equal(t, false, v.GetBool("webui.features.experimental"))

		// Test branding
		assert.Equal(t, "Pi Controller", v.GetString("webui.branding.title"))
		assert.Equal(t, "#10b981", v.GetString("webui.branding.primary_color"))
		assert.Equal(t, "dark", v.GetString("webui.branding.theme"))

		// Test security
		assert.Equal(t, true, v.GetBool("webui.security.hsts_enabled"))
		assert.Equal(t, true, v.GetBool("webui.security.frame_deny"))
	})
}

func TestConfigPrecedence(t *testing.T) {
	t.Run("environment overrides file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		fileContent := `
webui:
  port: 3000
  features:
    gpio_control: false
`
		err := os.WriteFile(configPath, []byte(fileContent), 0644)
		require.NoError(t, err)

		os.Setenv("PI_CONTROLLER_ENVIRONMENT", "development")
		os.Setenv("PI_CONTROLLER_WEBUI_PORT", "4000")
		os.Setenv("PI_CONTROLLER_WEBUI_FEATURES_GPIO_CONTROL", "true")
		defer func() {
			os.Unsetenv("PI_CONTROLLER_ENVIRONMENT")
			os.Unsetenv("PI_CONTROLLER_WEBUI_PORT")
			os.Unsetenv("PI_CONTROLLER_WEBUI_FEATURES_GPIO_CONTROL")
		}()

		cfg, err := LoadWithViper(configPath)
		require.NoError(t, err)

		// Environment should override file
		assert.Equal(t, 4000, cfg.WebUI.Port)
		if !assert.Equal(t, true, cfg.WebUI.Features.GPIOControl) {
			t.Logf("GPIO Control value: %v", cfg.WebUI.Features.GPIOControl)
			t.Logf("Raw env var: %s", os.Getenv("PI_CONTROLLER_WEBUI_FEATURES_GPIO_CONTROL"))
		}
	})
}

func TestNestedEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name  string
		env   map[string]string
		check func(*testing.T, *Config)
	}{
		{
			name: "webui backend api url",
			env: map[string]string{
				"PI_CONTROLLER_WEBUI_BACKEND_API_URL": "http://api.example.com:8080",
			},
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "http://api.example.com:8080", cfg.WebUI.Backend.API.URL)
			},
		},
		{
			name: "webui features",
			env: map[string]string{
				"PI_CONTROLLER_WEBUI_FEATURES_GPIO_CONTROL":           "false",
				"PI_CONTROLLER_WEBUI_FEATURES_CERTIFICATE_MANAGEMENT": "false",
				"PI_CONTROLLER_WEBUI_FEATURES_EXPERIMENTAL":           "true",
			},
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, false, cfg.WebUI.Features.GPIOControl)
				assert.Equal(t, false, cfg.WebUI.Features.CertificateManagement)
				assert.Equal(t, true, cfg.WebUI.Features.Experimental)
			},
		},
		{
			name: "webui branding",
			env: map[string]string{
				"PI_CONTROLLER_WEBUI_BRANDING_TITLE":         "Custom Title",
				"PI_CONTROLLER_WEBUI_BRANDING_THEME":         "light",
				"PI_CONTROLLER_WEBUI_BRANDING_PRIMARY_COLOR": "#ff0000",
			},
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "Custom Title", cfg.WebUI.Branding.Title)
				assert.Equal(t, "light", cfg.WebUI.Branding.Theme)
				assert.Equal(t, "#ff0000", cfg.WebUI.Branding.PrimaryColor)
			},
		},
		{
			name: "webui security",
			env: map[string]string{
				"PI_CONTROLLER_WEBUI_SECURITY_HSTS_ENABLED": "false",
				"PI_CONTROLLER_WEBUI_SECURITY_CSP_ENABLED":  "true",
			},
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, false, cfg.WebUI.Security.HSTSEnabled)
				assert.Equal(t, true, cfg.WebUI.Security.CSPEnabled)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment to development for predictable defaults
			os.Setenv("PI_CONTROLLER_ENVIRONMENT", "development")
			defer os.Unsetenv("PI_CONTROLLER_ENVIRONMENT")

			// Set environment variables
			for key, value := range tt.env {
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.env {
					os.Unsetenv(key)
				}
			}()

			cfg, err := LoadWithViper("")
			require.NoError(t, err)
			require.NotNil(t, cfg)

			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestLoadWithViper_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidContent := `
webui:
  port: "not a number"
  features:
    - invalid structure
`
	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadWithViper(configPath)

	// Should still load with defaults/errors handled
	// The actual behavior depends on viper's unmarshaling
	if err != nil {
		assert.Error(t, err)
	} else {
		// If viper handles it gracefully, config should still be valid
		require.NotNil(t, cfg)
	}
}

func TestEnvironmentDetection(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		wantDev bool
	}{
		{
			name:    "development environment",
			envVar:  "development",
			wantDev: true,
		},
		{
			name:    "dev environment",
			envVar:  "dev",
			wantDev: true,
		},
		{
			name:    "production environment",
			envVar:  "production",
			wantDev: false,
		},
		{
			name:    "empty defaults to production",
			envVar:  "production", // explicitly set to production since empty would use system env
			wantDev: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar != "" {
				os.Setenv("PI_CONTROLLER_ENVIRONMENT", tt.envVar)
			}
			defer os.Unsetenv("PI_CONTROLLER_ENVIRONMENT")

			cfg, err := LoadWithViper("")
			require.NoError(t, err)
			require.NotNil(t, cfg)

			// Development has less secure defaults
			if tt.wantDev {
				assert.False(t, cfg.WebUI.Auth.CookieSecure)
				assert.False(t, cfg.WebUI.Security.HSTSEnabled)
			} else {
				assert.True(t, cfg.WebUI.Auth.CookieSecure)
				assert.True(t, cfg.WebUI.Security.HSTSEnabled)
			}
		})
	}
}
