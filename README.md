# dmcheck

Languages: English | [简体中文](README.zh-CN.md)

**[dmcheck.app](https://dmcheck.app)** — Free domain availability checker powered by WHOIS & RDAP.

Enter a keyword, instantly check whether domains across multiple TLDs are available — results stream back in real time.

## Features

- **Real-time streaming** — results appear one by one via SSE, no waiting for all queries to finish
- **WHOIS + RDAP** — direct queries to registry WHOIS servers (port 43) with RDAP fallback via IANA bootstrap
- **83 TLDs** preconfigured with WHOIS servers; 1000+ TLDs supported via RDAP fallback
- **Customizable TLD list** — users can edit their TLD list in-browser (saved to localStorage)
- **Domain detail panel** — registration dates, registrar, DNS servers, status codes, raw WHOIS, optional site screenshot & favicon
- **Optional registrar links and price comparison** — when enabled, available domains can show configured registrar links, first-year USD reference prices, and a detail-drawer comparison
- **Reserved domain detection** — identifies registry-reserved domains separately from registered or available
- **Multi-language** — English (default), 中文, 日本語, 한국어, Español
- **Redis caching** — optional; gracefully degrades to no-cache mode
- **Rate limiting** — IP-based token bucket to prevent abuse
- **Single binary** — all static assets embedded via `go:embed`

## Quick Start

### Prerequisites

- Go 1.21+
- (Optional) Node.js 18+ for refreshing registrar price references
- (Optional) Redis for caching

### Run locally

```bash
go run .
```

The server starts on `http://localhost:3300` by default.

### Build

```bash
go build -o dmcheck .
./dmcheck
```

## Configuration

### Config files


| File                        | Description                     |
| --------------------------- | ------------------------------- |
| `config/whois-servers.json` | TLD → WHOIS server mapping      |
| `config/default-tlds.json`  | Default TLD list shown to users |
| `config/registrar-prices.json` | Optional registrar link templates and configured TLD price references; used only when `REGISTRAR_PRICES_ENABLED=true` |


### Registrar links and price comparison

dmcheck can run as a plain domain availability checker with no registrar data. This is the default mode and only requires Go. In this mode, `config/registrar-prices.json` is ignored, API responses omit `registration_options`, and the UI does not show registrar actions or price comparison.

Set `REGISTRAR_PRICES_ENABLED=true` to enable the optional registrar module. When enabled, available-domain results include registrar search/register links, first-year USD reference prices where available, and the comparison drawer in the UI.

`config/registrar-prices.json` contains two related but separate things: enabled registrar channels for outbound registration/search links, and automated price rows for the comparison UI. A registrar can be available as a link-only channel even when it is not used for automated pricing.

Current coverage as of `config/registrar-prices.json` `updated_at=2026-05-15`: 5 enabled registrar channels, 833 priced TLDs total, including 204 multi-label TLDs.

| Registrar | User-facing channel | Automated price rows | Update mode | Notes |
| --------- | ------------------- | -------------------- | ----------- | ----- |
| Cloudflare Registrar | Registration dashboard link | 0 | Link only | Cloudflare publishes registrar positioning and supported TLD information, but not a broad public price table suitable for unattended refreshes. |
| Porkbun | Registration/search link and price comparison | 632 | Automated from official public pricing page | Used for broad TLD coverage. |
| Namecheap | Registration/search link | 0 | Link only | Public pricing pages block scripted refresh and the official pricing API requires account credentials. |
| Spaceship | Registration/search link | 0 | Link only | A public domain pricing page exists at `https://www.spaceship.com/domains/` and exposes more rows through a "See more" UI, but the unattended updater receives Cloudflare verification from Node, curl, and headless Chrome. The official API also requires key/secret credentials, so this stays link-only until there is a stable non-interactive source. |
| Dynadot | Registration/search link and price comparison | 810 | Automated from official public pricing page | Used for broad TLD coverage, including many multi-label TLDs. |

NameSilo and GoDaddy have been evaluated but are not enabled channels in the current config. They are kept out until we have a stable unattended source or an intentional credential-backed integration.

### Updating registrar price references

```bash
node scripts/update-registrar-prices.mjs --date=YYYY-MM-DD
```

Update mechanism:

- Run the script from the repo root weekly and before each release that changes registrar behavior.
- The script fetches the official Porkbun and Dynadot pricing pages, parses first-year registration and renewal prices, removes price rows from non-automated sources, sorts the generated data, and writes `config/registrar-prices.json`.
- The script refuses to overwrite the config if Porkbun returns fewer than 100 rows or Dynadot returns fewer than 300 rows, which helps avoid replacing the price table with an anti-bot or transient error page.
- Review the printed coverage counts after every refresh, then commit the generated `config/registrar-prices.json` only when both automated sources pass their guards.

If a source starts failing consistently, leave it out of automated pricing until it has a stable public feed again.

Providers intentionally not automated or priced: Namecheap, NameSilo, Spaceship, and GoDaddy require account/API credentials, browser-only interaction, or block scripted access to public pricing pages; Cloudflare publishes registrar positioning and supported TLD information but not a broad public price table suitable for unattended refreshes.


### Environment variables

All runtime environment variables are read in `config.go` and stored in the global `AppConfig`. Other packages should read configuration from `AppConfig` instead of calling `os.Getenv` directly.

For systemd deployments, keep overrides in one server-side file: `/opt/dmcheck/dmcheck.env`. The provided `deploy/dmcheck.service` loads it through `EnvironmentFile`. This file is not committed or deployed by GitHub Actions, so create it once on the server or copy a local private copy securely.

| Variable     | Default          | Description                                                  |
| ------------ | ---------------- | ------------------------------------------------------------ |
| `PORT`       | `3300`           | HTTP server port                                             |
| `REDIS_ADDR` | `localhost:6379` | Redis address (set empty to disable)                         |
| `RATE_LIMIT` | `2`              | Requests per second per IP                                   |
| `RATE_BURST` | `5`              | Rate limiter burst size                                      |
| `AVAILABLE_CACHE_TTL` | `0`              | Cache TTL for available domains; `0` disables available-result caching |
| `REGISTERED_CACHE_TTL` | `2160h`          | Max cache TTL for registered/reserved domains; registered domains are refreshed before expiry |
| `CACHE_TTL`  | (empty)          | Legacy alias for `AVAILABLE_CACHE_TTL`                       |
| `REGISTRAR_PRICES_ENABLED` | `false`          | Enables registrar links and price comparison from `config/registrar-prices.json` |
| `GA_ID`      | (empty)          | Google Analytics Measurement ID (omit to disable)            |
| `SITE_URL`   | `https://dmcheck.app` | Canonical site URL used in rendered SEO metadata             |
| `APP_VERSION` | build metadata  | Optional fixed version string shown in the UI; leave empty to use build/git metadata |


## Project Structure

```
├── main.go              # Entry point, routing, middleware
├── whois.go             # WHOIS/RDAP query logic
├── handlers.go          # HTTP handlers (search, whois API)
├── ratelimit.go         # IP-based rate limiter
├── config.go            # Configuration and environment loading
├── config/
│   ├── whois-servers.json
│   ├── default-tlds.json
│   └── registrar-prices.json
├── scripts/
│   └── update-registrar-prices.mjs
├── static/
│   ├── index.html       # Main page
│   ├── app.js           # Frontend logic + i18n
│   ├── style.css
│   ├── lang/            # Translation files
│   │   ├── en.json
│   │   ├── zh.json
│   │   ├── ja.json
│   │   ├── ko.json
│   │   └── es.json
│   ├── tos.html
│   ├── privacy.html
│   └── legal.css
└── deploy/              # systemd, env-file & nginx configs
```

## Deployment

### 1. VPS prerequisites

- Ubuntu 20.04+ with Nginx installed
- Domain DNS A record pointing to VPS IP
- (Optional) Redis for caching

### 2. First-time server setup

```bash
# Create application directory and server-side env file
sudo mkdir -p /opt/dmcheck
sudo chown deployer:deployer /opt/dmcheck
sudo install -m 0644 deploy/dmcheck.env.example /opt/dmcheck/dmcheck.env
sudo chown deployer:deployer /opt/dmcheck/dmcheck.env

# Build locally and upload (or let GitHub Actions do it)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dmcheck .
scp dmcheck your-user@your-vps:/opt/dmcheck/

# Install systemd service
sudo cp deploy/dmcheck.service /etc/systemd/system/dmcheck.service
sudo systemctl daemon-reload
sudo systemctl enable dmcheck
sudo systemctl start dmcheck
```

### 3. Nginx + HTTPS

```bash
# Copy HTTP-only config and enable
sudo cp deploy/dmcheck.nginx /etc/nginx/sites-available/dmcheck
sudo ln -s /etc/nginx/sites-available/dmcheck /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx

# Issue certificate — certbot auto-adds SSL config to Nginx
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d dmcheck.app
```

Certbot will modify the Nginx config to add SSL listeners, redirect HTTP→HTTPS, and set up auto-renewal.

### 5. GitHub Actions auto-deploy

The workflow in `.github/workflows/deploy.yml` automatically builds and deploys on every push to `publish`.

Add these secrets in your GitHub repo (Settings → Secrets → Actions):


| Secret        | Value                      |
| ------------- | -------------------------- |
| `VPS_HOST`    | Your VPS IP or hostname    |
| `VPS_PORT`    | SSH port (if not 22)       |
| `VPS_USER`    | `deployer`                 |
| `VPS_SSH_KEY` | Private key for SSH access |


Flow: `git push origin publish` → GitHub Actions builds `linux/amd64` binary → `scp` to VPS → `systemctl restart dmcheck`. The workflow deploys the binary only; it does not upload or overwrite `/opt/dmcheck/dmcheck.env`.

### 6. Environment variables on VPS

Edit the environment file loaded by systemd. This is a persistent server-side file, not a committed repo file:

```bash
sudo nano /opt/dmcheck/dmcheck.env
```

```env
RATE_LIMIT=5
RATE_BURST=10
REGISTERED_CACHE_TTL=2160h
REGISTRAR_PRICES_ENABLED=true
GA_ID=G-XXXXXXXXXX
SITE_URL=https://dmcheck.app
# Leave empty to use the build/git-derived version.
APP_VERSION=
```

Then run:

```bash
sudo systemctl daemon-reload
sudo systemctl restart dmcheck
```

## Acknowledgments

- [IANA](https://data.iana.org/rdap/dns.json) — RDAP bootstrap data
- [rdap.org](https://rdap.org) — RDAP query fallback
- [screenshot.domains](https://screenshot.domains) — website screenshots
- [favicon.im](https://favicon.im) — website favicons
- [Porkbun domain pricing](https://porkbun.com/products/domains/) and [Dynadot domain pricing](https://www.dynadot.com/domain/prices) are used as registration price references

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on adding translations, WHOIS servers, and submitting pull requests.

## License

[AGPL-3.0](LICENSE)
