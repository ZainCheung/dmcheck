package main

import (
	"embed"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

//go:embed config/*
var configFiles embed.FS

type runtimeConfig struct {
	Port                   string
	RedisAddr              string
	RateLimit              rate.Limit
	RateBurst              int
	AvailableCacheTTL      time.Duration
	RegisteredCacheTTL     time.Duration
	RegistrarPricesEnabled bool
	GAID                   string
	SiteURL                string
	AppVersion             string
}

var AppConfig = defaultRuntimeConfig()

func defaultRuntimeConfig() runtimeConfig {
	return runtimeConfig{
		Port:               "3300",
		RedisAddr:          "localhost:6379",
		RateLimit:          rate.Limit(2),
		RateBurst:          5,
		RegisteredCacheTTL: 90 * 24 * time.Hour,
		SiteURL:            "https://dmcheck.app",
	}
}

func loadConfig() {
	AppConfig = loadEnvConfig()
	loadWhoisServers()
	loadDefaultTLDs()
	loadRegistrarPrices()
}

func loadWhoisServers() {
	data, err := readConfigFile("config/whois-servers.json")
	if err != nil {
		log.Printf("Using embedded whois-servers.json: %v", err)
		data, err = configFiles.ReadFile("config/whois-servers.json")
		if err != nil {
			log.Fatalf("Cannot read embedded whois-servers.json: %v", err)
		}
	}
	var servers map[string]string
	if err := json.Unmarshal(data, &servers); err != nil {
		log.Fatalf("Invalid whois-servers.json: %v", err)
	}
	whoisServers = servers
	log.Printf("Loaded %d WHOIS servers", len(whoisServers))
}

func loadDefaultTLDs() {
	data, err := readConfigFile("config/default-tlds.json")
	if err != nil {
		log.Printf("Using embedded default-tlds.json: %v", err)
		data, err = configFiles.ReadFile("config/default-tlds.json")
		if err != nil {
			log.Fatalf("Cannot read embedded default-tlds.json: %v", err)
		}
	}
	var tlds []string
	if err := json.Unmarshal(data, &tlds); err != nil {
		log.Fatalf("Invalid default-tlds.json: %v", err)
	}
	DefaultTLDs = tlds
	log.Printf("Loaded %d default TLDs", len(DefaultTLDs))
}

func loadEnvConfig() runtimeConfig {
	cfg := defaultRuntimeConfig()
	cfg.Port = envString("PORT", cfg.Port)
	cfg.RedisAddr = envStringAllowEmpty("REDIS_ADDR", cfg.RedisAddr)
	cfg.RateLimit = rate.Limit(envFloat("RATE_LIMIT", float64(cfg.RateLimit)))
	cfg.RateBurst = envInt("RATE_BURST", cfg.RateBurst)
	cfg.AvailableCacheTTL = envDuration("CACHE_TTL", cfg.AvailableCacheTTL)
	cfg.AvailableCacheTTL = envDuration("AVAILABLE_CACHE_TTL", cfg.AvailableCacheTTL)
	cfg.RegisteredCacheTTL = envDuration("REGISTERED_CACHE_TTL", cfg.RegisteredCacheTTL)
	cfg.RegistrarPricesEnabled = envBool("REGISTRAR_PRICES_ENABLED", cfg.RegistrarPricesEnabled)
	cfg.GAID = envString("GA_ID", cfg.GAID)
	cfg.SiteURL = strings.TrimRight(envString("SITE_URL", cfg.SiteURL), "/")
	cfg.AppVersion = envString("APP_VERSION", cfg.AppVersion)
	return cfg
}

func envString(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envStringAllowEmpty(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(v)
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
		log.Printf("Invalid %s=%q; expected number", key, v)
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
		log.Printf("Invalid %s=%q; expected integer", key, v)
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		log.Printf("Invalid %s=%q; expected Go duration", key, v)
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
		log.Printf("Invalid %s=%q; expected true or false", key, v)
	}
	return fallback
}

func readConfigFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
