# dmcheck

Languages: English | [简体中文](README.zh-CN.md)

**[dmcheck.app](https://dmcheck.app)** — Free domain availability checker powered by WHOIS & RDAP.

Enter a keyword, instantly check whether domains across multiple TLDs are available — results stream back in real time.

## Product Experience Principles

dmcheck is designed for fast, repeated domain screening: enter one keyword, scan a focused set of TLD results, inspect details only when needed, and optionally create a compact result image for sharing.

- **Feels fast first** — the interface stays quiet, results stream in place, cached data is used where safe, and heavy detail queries happen only after a row is opened.
- **Simple input, precise control** — the primary path is a single keyword input; TLD settings use a long text area so users can paste, edit, remove, and reorder suffixes without checkbox fatigue.
- **Scannable results** — available domains are grouped first after completion, each row shows the domain and one clear result, registered domains show the registration date, and very recent registrations use relative labels such as today, yesterday, or N days ago.
- **Details stay out of the way** — registered, reserved, and unknown domains open a detail drawer with cached preview first and live WHOIS/RDAP details after click; website screenshot and favicon loading remain on demand.
- **Sharing is a separate flow** — the main page width is optimized for use, not screenshots. The result image flow generates a narrow PNG, previews it in a dialog, then offers copy, download, and system share actions.
- **Homepage stays focused** — FAQ content lives on a dedicated page while homepage SEO is handled through metadata, structured data, canonical/hreflang links, robots, and sitemap files.
- **Efficient visual tone** — the UI should feel minimal, technical, and quick, with Graphite/Carbon themes, restrained motion, no broad gradients, and no decorative distractions.
- **Debuggable in production** — the footer and `app-version` meta tag expose the deployed version so bug reports can be tied to a specific build.

## Features

- **Real-time streaming** — results appear one by one via SSE, no waiting for all queries to finish
- **WHOIS + RDAP** — direct queries to registry WHOIS servers (port 43) with RDAP fallback via IANA bootstrap
- **83 TLDs** preconfigured with WHOIS servers; 1000+ TLDs supported via RDAP fallback
- **Customizable TLD list** — users can edit their TLD list in-browser (saved to localStorage)
- **Domain detail panel** — registration dates, registrar, DNS servers, status codes, raw WHOIS, optional site screenshot & favicon
- **Registrar links and price comparison** — available domains can show configured registrar links, first-year USD reference prices, and a detail-drawer comparison
- **Result image export** — compact PNG preview with copy, download, and system share actions
- **Reserved domain detection** — identifies registry-reserved domains separately from registered or available
- **Multi-language** — English (default), 中文, 日本語, 한국어, Español
- **Redis caching** — optional; gracefully degrades to no-cache mode
- **Rate limiting** — IP-based token bucket to prevent abuse
- **Single binary** — all static assets embedded via `go:embed`

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+ for refreshing registrar price references
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
| `config/registrar-prices.json` | Registrar link templates and configured TLD price references |


### Updating registrar price references

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


| Variable     | Default          | Description                                                  |
| ------------ | ---------------- | ------------------------------------------------------------ |
| `PORT`       | `3300`           | HTTP server port                                             |
| `REDIS_ADDR` | `localhost:6379` | Redis address (set empty to disable)                         |
| `RATE_LIMIT` | `2`              | Requests per second per IP                                   |
| `RATE_BURST` | `5`              | Rate limiter burst size                                      |
| `AVAILABLE_CACHE_TTL` | `0`              | Cache TTL for available domains; `0` disables available-result caching |
| `REGISTERED_CACHE_TTL` | `2160h`          | Max cache TTL for registered/reserved domains; registered domains are refreshed before expiry |
| `CACHE_TTL`  | (empty)          | Legacy alias for `AVAILABLE_CACHE_TTL`                       |
| `GA_ID`      | (empty)          | Google Analytics Measurement ID (omit to disable)            |


## Project Structure

```
├── main.go              # Entry point, routing, middleware
├── whois.go             # WHOIS/RDAP query logic
├── handlers.go          # HTTP handlers (search, whois API)
├── ratelimit.go         # IP-based rate limiter
├── config.go            # Configuration loading
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
└── deploy/              # systemd & nginx configs
```

## Deployment

### 1. VPS prerequisites

- Ubuntu 20.04+ with Nginx installed
- Domain DNS A record pointing to VPS IP
- (Optional) Redis for caching

### 2. First-time server setup

```bash
# Create application directory
sudo mkdir -p /opt/dmcheck
sudo chown deployer:deployer /opt/dmcheck

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


Flow: `git push origin publish` → GitHub Actions builds `linux/amd64` binary → `scp` to VPS → `systemctl restart dmcheck`.

### 6. Environment variables on VPS

Edit the systemd service to add environment variables:

```bash
sudo systemctl edit dmcheck
```

```ini
[Service]
Environment=RATE_LIMIT=5
Environment=RATE_BURST=10
Environment=REGISTERED_CACHE_TTL=2160h
Environment=GA_ID=G-XXXXXXXXXX
```

Then `sudo systemctl restart dmcheck`.

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
