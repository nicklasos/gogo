package unit

import (
	"testing"

	"app/config"

	"github.com/stretchr/testify/assert"
)

func TestConfig_ValidateJWTSecret(t *testing.T) {
	strong := "3f9c1e7a5b2d4c6e8f0a1b3c5d7e9f1a2b4c6d8e"

	cases := []struct {
		name    string
		env     string
		secret  string
		wantErr bool
		weak    bool
	}{
		{"empty is always refused", "development", "", true, true},
		{"blank is always refused", "development", "   ", true, true},
		{"short is tolerated in development", "development", "dev-secret", false, true},
		{"short is refused in production", "production", "dev-secret", true, true},
		{"the .env.example value is refused in production", "production", "your-super-secret-jwt-key-change-this-in-production", true, true},
		{"the .env.example value is tolerated in development", "development", "your-super-secret-jwt-key-change-this-in-production", false, true},
		{"a long random secret passes in production", "production", strong, false, false},
		{"production is matched case-insensitively", "Production", "dev-secret", true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{Environment: tc.env, JWTSecret: tc.secret}
			err := cfg.ValidateJWTSecret()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.weak, cfg.JWTSecretIsWeak())
		})
	}
}

func TestConfig_LoadAuthDefaults(t *testing.T) {
	for _, key := range []string{"JWT_ACCESS_TOKEN_TTL", "ALLOW_REGISTRATION", "AUTH_RATE_LIMIT", "TRUSTED_PROXIES", "FRONTEND_URL"} {
		t.Setenv(key, "")
	}
	cfg, err := config.Load()
	assert.NoError(t, err)
	assert.Equal(t, "1h0m0s", cfg.AccessTokenTTL.String())
	assert.False(t, cfg.AllowRegistration)
	assert.True(t, cfg.AuthRateLimit)
	assert.Equal(t, []string{"127.0.0.1", "::1"}, cfg.TrustedProxies)

	t.Setenv("JWT_ACCESS_TOKEN_TTL", "15m")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.1, 10.0.0.2")
	t.Setenv("FRONTEND_URL", "https://admin.example.com/")
	cfg, err = config.Load()
	assert.NoError(t, err)
	assert.Equal(t, "15m0s", cfg.AccessTokenTTL.String())
	assert.Equal(t, []string{"10.0.0.1", "10.0.0.2"}, cfg.TrustedProxies)
	assert.Equal(t, "https://admin.example.com", cfg.FrontendURL)

	t.Setenv("JWT_ACCESS_TOKEN_TTL", "soon")
	cfg, _ = config.Load()
	assert.Equal(t, "1h0m0s", cfg.AccessTokenTTL.String(), "an unparsable duration falls back to the default")
}
