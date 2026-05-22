package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strings"
)

type RegistrationOption struct {
	RegistrarID     string  `json:"registrar_id"`
	RegistrarName   string  `json:"registrar_name"`
	RegistrationUSD float64 `json:"registration_usd,omitempty"`
	RenewalUSD      float64 `json:"renewal_usd,omitempty"`
	RegisterURL     string  `json:"register_url"`
	SourceURL       string  `json:"source_url,omitempty"`
	UpdatedAt       string  `json:"updated_at,omitempty"`
	IsLowest        bool    `json:"is_lowest,omitempty"`
	Sponsored       bool    `json:"sponsored,omitempty"`
}

type registrarPriceConfig struct {
	Currency   string                            `json:"currency"`
	UpdatedAt  string                            `json:"updated_at"`
	Registrars []registrarConfig                 `json:"registrars"`
	Prices     map[string]map[string]tldPriceRow `json:"prices"`
}

type registrarConfig struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Enabled           bool   `json:"enabled"`
	SearchURLTemplate string `json:"search_url_template"`
	TrackingQuery     string `json:"tracking_query,omitempty"`
	SourceURL         string `json:"source_url,omitempty"`
}

type tldPriceRow struct {
	RegistrationUSD float64 `json:"registration_usd,omitempty"`
	RenewalUSD      float64 `json:"renewal_usd,omitempty"`
	SourceURL       string  `json:"source_url,omitempty"`
}

type registrationCatalog struct {
	Currency   string
	UpdatedAt  string
	Order      []string
	Registrars map[string]registrarConfig
	Prices     map[string]map[string]tldPriceRow
}

var registrationData registrationCatalog

func loadRegistrarPrices() {
	data, err := readConfigFile("config/registrar-prices.json")
	if err != nil {
		data, err = configFiles.ReadFile("config/registrar-prices.json")
		if err != nil {
			log.Printf("Registrar price config not available: %v", err)
			return
		}
	}

	var cfg registrarPriceConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("Invalid registrar-prices.json: %v", err)
		return
	}
	if err := configureRegistrationCatalog(cfg); err != nil {
		log.Printf("Invalid registrar price config: %v", err)
		return
	}
	log.Printf("Loaded %d registrars with prices for %d TLDs", len(registrationData.Order), len(registrationData.Prices))
}

func configureRegistrationCatalog(cfg registrarPriceConfig) error {
	catalog := registrationCatalog{
		Currency:   strings.ToUpper(strings.TrimSpace(cfg.Currency)),
		UpdatedAt:  strings.TrimSpace(cfg.UpdatedAt),
		Order:      make([]string, 0, len(cfg.Registrars)),
		Registrars: make(map[string]registrarConfig, len(cfg.Registrars)),
		Prices:     make(map[string]map[string]tldPriceRow, len(cfg.Prices)),
	}
	if catalog.Currency == "" {
		catalog.Currency = "USD"
	}
	if catalog.Currency != "USD" {
		return fmt.Errorf("unsupported registrar price currency %q", catalog.Currency)
	}

	for _, registrar := range cfg.Registrars {
		registrar.ID = normalizeDomain(registrar.ID)
		registrar.Name = strings.TrimSpace(registrar.Name)
		registrar.SearchURLTemplate = strings.TrimSpace(registrar.SearchURLTemplate)
		registrar.SourceURL = strings.TrimSpace(registrar.SourceURL)
		registrar.TrackingQuery = strings.TrimSpace(registrar.TrackingQuery)
		if registrar.ID == "" || registrar.Name == "" || registrar.SearchURLTemplate == "" {
			continue
		}
		catalog.Order = append(catalog.Order, registrar.ID)
		catalog.Registrars[registrar.ID] = registrar
	}

	for tld, rows := range cfg.Prices {
		tld = normalizeTLD(tld)
		if tld == "" {
			continue
		}
		catalog.Prices[tld] = make(map[string]tldPriceRow, len(rows))
		for registrarID, row := range rows {
			registrarID = normalizeDomain(registrarID)
			if _, ok := catalog.Registrars[registrarID]; !ok {
				continue
			}
			if row.RegistrationUSD <= 0 {
				continue
			}
			row.SourceURL = strings.TrimSpace(row.SourceURL)
			catalog.Prices[tld][registrarID] = row
		}
		if len(catalog.Prices[tld]) == 0 {
			delete(catalog.Prices, tld)
		}
	}

	registrationData = catalog
	return nil
}

func addRegistrationOptions(result DomainResult) DomainResult {
	if result.Status != "available" {
		result.RegistrationOptions = nil
		return result
	}
	result.RegistrationOptions = registrationOptionsForDomain(result.Domain)
	return result
}

func addRegistrationOptionsToResults(results []DomainResult) {
	for i := range results {
		results[i] = addRegistrationOptions(results[i])
	}
}

func registrationOptionsForDomain(domain string) []RegistrationOption {
	domain = normalizeDomain(domain)
	tld := domainTLD(domain)
	if domain == "" || tld == "" {
		return nil
	}

	rows := registrationData.Prices[tld]
	options := make([]RegistrationOption, 0, len(registrationData.Order))
	if len(rows) > 0 {
		for _, registrarID := range registrationData.Order {
			registrar, ok := registrationData.Registrars[registrarID]
			if !ok || !registrar.Enabled {
				continue
			}
			price, ok := rows[registrarID]
			if !ok {
				continue
			}
			options = append(options, registrationOption(domain, registrar, price))
		}
		sort.SliceStable(options, func(i, j int) bool {
			if options[i].RegistrationUSD == options[j].RegistrationUSD {
				return options[i].RegistrarName < options[j].RegistrarName
			}
			return options[i].RegistrationUSD < options[j].RegistrationUSD
		})
		if len(options) > 0 {
			options[0].IsLowest = true
		}
		return options
	}

	for _, registrarID := range registrationData.Order {
		registrar, ok := registrationData.Registrars[registrarID]
		if !ok || !registrar.Enabled {
			continue
		}
		options = append(options, registrationOption(domain, registrar, tldPriceRow{}))
	}
	return options
}

func registrationOption(domain string, registrar registrarConfig, price tldPriceRow) RegistrationOption {
	sourceURL := price.SourceURL
	if sourceURL == "" {
		sourceURL = registrar.SourceURL
	}
	return RegistrationOption{
		RegistrarID:     registrar.ID,
		RegistrarName:   registrar.Name,
		RegistrationUSD: price.RegistrationUSD,
		RenewalUSD:      price.RenewalUSD,
		RegisterURL:     registrarURL(domain, registrar),
		SourceURL:       sourceURL,
		UpdatedAt:       registrationData.UpdatedAt,
		Sponsored:       registrar.TrackingQuery != "",
	}
}

func registrarURL(domain string, registrar registrarConfig) string {
	tld := domainTLD(domain)
	sld := domainSLD(domain, tld)
	out := strings.ReplaceAll(registrar.SearchURLTemplate, "{domain}", url.QueryEscape(domain))
	out = strings.ReplaceAll(out, "{sld}", url.QueryEscape(sld))
	out = strings.ReplaceAll(out, "{tld}", url.QueryEscape(tld))
	if registrar.TrackingQuery == "" {
		return out
	}
	parsed, err := url.Parse(out)
	if err != nil {
		sep := "?"
		if strings.Contains(out, "?") {
			sep = "&"
		}
		return out + sep + registrar.TrackingQuery
	}
	extra, err := url.ParseQuery(registrar.TrackingQuery)
	if err != nil {
		return out
	}
	query := parsed.Query()
	for key, values := range extra {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func domainTLD(domain string) string {
	domain = normalizeDomain(domain)
	if domain == "" {
		return ""
	}

	best := ""
	for tld := range registrationData.Prices {
		if strings.HasSuffix(domain, "."+tld) && len(tld) > len(best) {
			best = tld
		}
	}
	if best != "" {
		return best
	}

	idx := strings.LastIndex(domain, ".")
	if idx < 0 || idx == len(domain)-1 {
		return ""
	}
	return normalizeTLD(domain[idx+1:])
}

func domainSLD(domain, tld string) string {
	if tld != "" {
		suffix := "." + tld
		if strings.HasSuffix(domain, suffix) {
			return strings.TrimSuffix(domain, suffix)
		}
	}
	if idx := strings.LastIndex(domain, "."); idx > 0 {
		return domain[:idx]
	}
	return domain
}

func normalizeTLD(tld string) string {
	return normalizeDomain(strings.TrimPrefix(tld, "."))
}
