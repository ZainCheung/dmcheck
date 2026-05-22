package main

import (
	"testing"
	"time"
)

func TestCacheTTLPolicy(t *testing.T) {
	oldConfig := AppConfig
	defer func() { AppConfig = oldConfig }()

	AppConfig.AvailableCacheTTL = 0
	AppConfig.RegisteredCacheTTL = 90 * 24 * time.Hour
	now := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		result DomainResult
		want   time.Duration
		ok     bool
	}{
		{
			name:   "available disabled by default",
			result: DomainResult{Status: "available"},
			ok:     false,
		},
		{
			name:   "unknown is not cached",
			result: DomainResult{Status: "unknown"},
			ok:     false,
		},
		{
			name:   "reserved uses registered ttl",
			result: DomainResult{Status: "reserved"},
			want:   90 * 24 * time.Hour,
			ok:     true,
		},
		{
			name:   "registered caps at ninety days",
			result: DomainResult{Status: "registered", Expires: now.Add(120 * 24 * time.Hour).Format(time.RFC3339)},
			want:   90 * 24 * time.Hour,
			ok:     true,
		},
		{
			name:   "registered refreshes before expiry",
			result: DomainResult{Status: "registered", Expires: now.Add(10 * 24 * time.Hour).Format(time.RFC3339)},
			want:   9 * 24 * time.Hour,
			ok:     true,
		},
		{
			name:   "registered near expiry falls back to one hour",
			result: DomainResult{Status: "registered", Expires: now.Add(12 * time.Hour).Format(time.RFC3339)},
			want:   time.Hour,
			ok:     true,
		},
		{
			name:   "registered without expiry uses one day",
			result: DomainResult{Status: "registered"},
			want:   24 * time.Hour,
			ok:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := cacheTTL(tt.result, now)
			if ok != tt.ok {
				t.Fatalf("cacheTTL ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("cacheTTL ttl = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAvailableCacheTTLCanBeEnabled(t *testing.T) {
	oldConfig := AppConfig
	defer func() { AppConfig = oldConfig }()

	AppConfig.AvailableCacheTTL = 15 * time.Minute
	got, ok := cacheTTL(DomainResult{Status: "available"}, time.Now())
	if !ok {
		t.Fatal("cacheTTL did not enable available caching")
	}
	if got != 15*time.Minute {
		t.Fatalf("cacheTTL ttl = %v, want %v", got, 15*time.Minute)
	}
}
