package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	t.Run("should load default values", func(t *testing.T) {
		cfg, err := Load("")
		assert.NoError(t, err)
		assert.Equal(t, 8080, cfg.API.Port)
		assert.Equal(t, "info", cfg.Log.Level)
	})

	t.Run("should load from environment variables", func(t *testing.T) {
		t.Setenv("PI_CONTROLLER_API_PORT", "9090")
		t.Setenv("PI_CONTROLLER_LOG_LEVEL", "debug")
		t.Setenv("PI_CONTROLLER_DEBUG", "true")
		t.Setenv("PI_CONTROLLER_DATA_DIR", "/tmp/pi-controller")
		t.Setenv("PI_CONTROLLER_API_HOST", "localhost")

		cfg, err := Load("")
		assert.NoError(t, err)
		assert.Equal(t, 9090, cfg.API.Port)
		assert.Equal(t, "debug", cfg.Log.Level)
		assert.True(t, cfg.App.Debug)
		assert.Equal(t, "/tmp/pi-controller", cfg.App.DataDir)
		assert.Equal(t, "localhost", cfg.API.Host)
	})

	t.Run("should load from config file", func(t *testing.T) {
		content := `
api:
  port: 8888
  host: "testhost"
log:
  level: "warn"
app:
  debug: true
  data_dir: "/tmp/test"
`
		tmpfile, err := os.CreateTemp("", "config-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		cfg, err := Load(tmpfile.Name())
		assert.NoError(t, err)
		assert.Equal(t, 8888, cfg.API.Port)
		assert.Equal(t, "warn", cfg.Log.Level)
		assert.True(t, cfg.App.Debug)
		assert.Equal(t, "/tmp/test", cfg.App.DataDir)
		assert.Equal(t, "testhost", cfg.API.Host)
	})

	t.Run("should override config file with environment variables", func(t *testing.T) {
		content := `
api:
  port: 8888
  host: "testhost"
log:
  level: "warn"
`
		tmpfile, err := os.CreateTemp("", "config-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		t.Setenv("PI_CONTROLLER_API_PORT", "9999")
		t.Setenv("PI_CONTROLLER_LOG_LEVEL", "panic")

		cfg, err := Load(tmpfile.Name())
		assert.NoError(t, err)
		assert.Equal(t, 9999, cfg.API.Port)
		assert.Equal(t, "panic", cfg.Log.Level)
		assert.Equal(t, "testhost", cfg.API.Host)
	})

	t.Run("should return an error for invalid config file", func(t *testing.T) {
		content := `
api:
  port: 8888
  host: "testhost"
log:
  level: "warn"
app:
  debug: true
  data_dir: "/tmp/test"
invalid-yaml
`
		tmpfile, err := os.CreateTemp("", "config-*.yaml")
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
