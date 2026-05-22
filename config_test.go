package main

import (
	"os"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func clearRuntimeEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"PORT",
		"REDIS_ADDR",
		"RATE_LIMIT",
		"RATE_BURST",
		"CACHE_TTL",
		"AVAILABLE_CACHE_TTL",
		"REGISTERED_CACHE_TTL",
		"REGISTRAR_PRICES_ENABLED",
		"GA_ID",
		"SITE_URL",
		"APP_VERSION",
	}
	for _, key := range keys {
		key := key
		value, ok := os.LookupEnv(key)
		os.Unsetenv(key)
		t.Cleanup(func() {
			if ok {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		})
	}
}

func TestLoadEnvConfigDefaults(t *testing.T) {
	clearRuntimeEnv(t)
	cfg := loadEnvConfig()

	if cfg.Port != "3300" {
		t.Fatalf("Port = %q, want 3300", cfg.Port)
	}
	if cfg.RedisAddr != "localhost:6379" {
		t.Fatalf("RedisAddr = %q, want localhost:6379", cfg.RedisAddr)
	}
	if cfg.RateLimit != rate.Limit(2) || cfg.RateBurst != 5 {
		t.Fatalf("rate config = %v/%d, want 2/5", cfg.RateLimit, cfg.RateBurst)
	}
	if cfg.AvailableCacheTTL != 0 {
		t.Fatalf("AvailableCacheTTL = %v, want 0", cfg.AvailableCacheTTL)
	}
	if cfg.RegisteredCacheTTL != 90*24*time.Hour {
		t.Fatalf("RegisteredCacheTTL = %v, want 2160h", cfg.RegisteredCacheTTL)
	}
	if cfg.RegistrarPricesEnabled {
		t.Fatalf("RegistrarPricesEnabled = true, want false")
	}
	if cfg.SiteURL != "https://dmcheck.app" {
		t.Fatalf("SiteURL = %q, want https://dmcheck.app", cfg.SiteURL)
	}
}

func TestLoadEnvConfigOverrides(t *testing.T) {
	clearRuntimeEnv(t)
	t.Setenv("PORT", "4400")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("RATE_LIMIT", "7.5")
	t.Setenv("RATE_BURST", "12")
	t.Setenv("CACHE_TTL", "3m")
	t.Setenv("AVAILABLE_CACHE_TTL", "9m")
	t.Setenv("REGISTERED_CACHE_TTL", "48h")
	t.Setenv("REGISTRAR_PRICES_ENABLED", "true")
	t.Setenv("GA_ID", "G-TEST")
	t.Setenv("SITE_URL", "https://example.test/")
	t.Setenv("APP_VERSION", "v-test")

	cfg := loadEnvConfig()

	if cfg.Port != "4400" {
		t.Fatalf("Port = %q, want 4400", cfg.Port)
	}
	if cfg.RedisAddr != "" {
		t.Fatalf("RedisAddr = %q, want empty string", cfg.RedisAddr)
	}
	if cfg.RateLimit != rate.Limit(7.5) || cfg.RateBurst != 12 {
		t.Fatalf("rate config = %v/%d, want 7.5/12", cfg.RateLimit, cfg.RateBurst)
	}
	if cfg.AvailableCacheTTL != 9*time.Minute {
		t.Fatalf("AvailableCacheTTL = %v, want 9m", cfg.AvailableCacheTTL)
	}
	if cfg.RegisteredCacheTTL != 48*time.Hour {
		t.Fatalf("RegisteredCacheTTL = %v, want 48h", cfg.RegisteredCacheTTL)
	}
	if !cfg.RegistrarPricesEnabled {
		t.Fatalf("RegistrarPricesEnabled = false, want true")
	}
	if cfg.GAID != "G-TEST" || cfg.SiteURL != "https://example.test" || cfg.AppVersion != "v-test" {
		t.Fatalf("site config = %+v", cfg)
	}
}
