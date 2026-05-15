package main

import "testing"

func TestRegistrationOptionsAreSortedAndMarked(t *testing.T) {
	old := registrationData
	defer func() { registrationData = old }()

	err := configureRegistrationCatalog(registrarPriceConfig{
		Currency:  "USD",
		UpdatedAt: "2026-05-15",
		Registrars: []registrarConfig{
			{ID: "slow", Name: "Slow Registrar", Enabled: true, SearchURLTemplate: "https://slow.example/search?domain={domain}", SourceURL: "https://slow.example/prices"},
			{ID: "fast", Name: "Fast Registrar", Enabled: true, SearchURLTemplate: "https://fast.example/register/{domain}", TrackingQuery: "utm_source=dmcheck"},
			{ID: "off", Name: "Disabled Registrar", Enabled: false, SearchURLTemplate: "https://off.example/{domain}"},
		},
		Prices: map[string]map[string]tldPriceRow{
			"com": {
				"slow": {RegistrationUSD: 12.50, RenewalUSD: 13.50},
				"fast": {RegistrationUSD: 8.99, RenewalUSD: 14.99},
				"off":  {RegistrationUSD: 1.00, RenewalUSD: 1.00},
			},
		},
	})
	if err != nil {
		t.Fatalf("configureRegistrationCatalog returned error: %v", err)
	}

	options := registrationOptionsForDomain("Example.COM")
	if len(options) != 2 {
		t.Fatalf("registrationOptionsForDomain returned %d options, want 2", len(options))
	}
	if options[0].RegistrarID != "fast" || !options[0].IsLowest {
		t.Fatalf("lowest option = %+v, want fast marked lowest", options[0])
	}
	if options[0].RegisterURL != "https://fast.example/register/example.com?utm_source=dmcheck" {
		t.Fatalf("register URL = %q", options[0].RegisterURL)
	}
	if options[0].UpdatedAt != "2026-05-15" {
		t.Fatalf("updated_at = %q", options[0].UpdatedAt)
	}
	if options[1].SourceURL != "https://slow.example/prices" {
		t.Fatalf("source_url = %q", options[1].SourceURL)
	}
}

func TestRegistrationOptionsFallbackToGenericLinks(t *testing.T) {
	old := registrationData
	defer func() { registrationData = old }()

	err := configureRegistrationCatalog(registrarPriceConfig{
		Currency:  "USD",
		UpdatedAt: "2026-05-15",
		Registrars: []registrarConfig{
			{ID: "one", Name: "One", Enabled: true, SearchURLTemplate: "https://one.example/?q={domain}"},
			{ID: "two", Name: "Two", Enabled: true, SearchURLTemplate: "https://two.example/?sld={sld}&tld={tld}"},
		},
	})
	if err != nil {
		t.Fatalf("configureRegistrationCatalog returned error: %v", err)
	}

	options := registrationOptionsForDomain("brand.im")
	if len(options) != 2 {
		t.Fatalf("registrationOptionsForDomain returned %d options, want 2", len(options))
	}
	if options[0].RegistrationUSD != 0 || options[0].IsLowest {
		t.Fatalf("generic option should not expose price or lowest flag: %+v", options[0])
	}
	if options[1].RegisterURL != "https://two.example/?sld=brand&tld=im" {
		t.Fatalf("template expansion = %q", options[1].RegisterURL)
	}
}

func TestRegistrationOptionsUseLongestPricedSuffix(t *testing.T) {
	old := registrationData
	defer func() { registrationData = old }()

	err := configureRegistrationCatalog(registrarPriceConfig{
		Currency: "USD",
		Registrars: []registrarConfig{
			{ID: "one", Name: "One", Enabled: true, SearchURLTemplate: "https://one.example/?sld={sld}&tld={tld}"},
		},
		Prices: map[string]map[string]tldPriceRow{
			"uk": {
				"one": {RegistrationUSD: 12.00, RenewalUSD: 12.00},
			},
			"co.uk": {
				"one": {RegistrationUSD: 8.00, RenewalUSD: 9.00},
			},
		},
	})
	if err != nil {
		t.Fatalf("configureRegistrationCatalog returned error: %v", err)
	}

	options := registrationOptionsForDomain("Brand.CO.UK")
	if len(options) != 1 {
		t.Fatalf("registrationOptionsForDomain returned %d options, want 1", len(options))
	}
	if options[0].RegistrationUSD != 8.00 {
		t.Fatalf("registration_usd = %.2f, want 8.00", options[0].RegistrationUSD)
	}
	if options[0].RegisterURL != "https://one.example/?sld=brand&tld=co.uk" {
		t.Fatalf("template expansion = %q", options[0].RegisterURL)
	}
}

func TestAddRegistrationOptionsOnlyForAvailableDomains(t *testing.T) {
	old := registrationData
	defer func() { registrationData = old }()

	err := configureRegistrationCatalog(registrarPriceConfig{
		Currency: "USD",
		Registrars: []registrarConfig{
			{ID: "one", Name: "One", Enabled: true, SearchURLTemplate: "https://one.example/?q={domain}"},
		},
	})
	if err != nil {
		t.Fatalf("configureRegistrationCatalog returned error: %v", err)
	}

	available := addRegistrationOptions(DomainResult{Domain: "brand.com", Status: "available"})
	if len(available.RegistrationOptions) != 1 {
		t.Fatalf("available domain got %d options, want 1", len(available.RegistrationOptions))
	}

	registered := addRegistrationOptions(DomainResult{Domain: "brand.com", Status: "registered"})
	if len(registered.RegistrationOptions) != 0 {
		t.Fatalf("registered domain got options: %+v", registered.RegistrationOptions)
	}
}
