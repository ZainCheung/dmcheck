# dmcheck

语言：[English](README.md) | 简体中文

**[dmcheck.app](https://dmcheck.app)** — 基于 WHOIS 和 RDAP 的免费域名可用性查询工具。

输入一个关键词，即可同时查询多个后缀下的域名是否可注册，结果会实时流式返回。

## 功能

- **实时流式结果** — 通过 SSE 逐条展示结果，不必等待所有查询完成
- **WHOIS + RDAP** — 直接查询注册局 WHOIS 服务器（43 端口），并通过 IANA bootstrap 回退到 RDAP
- **预置 83 个 TLD**，并可通过 RDAP fallback 支持 1000+ 后缀
- **可自定义 TLD 列表** — 用户可在浏览器内编辑后缀列表，并保存到 localStorage
- **域名详情面板** — 展示注册日期、注册商、DNS 服务器、状态码、原始 WHOIS，以及可选加载的网站截图和 favicon
- **注册商链接和价格比价** — 可注册域名可展示注册商跳转链接、首年美元参考价，以及详情抽屉中的注册商比价
- **保留域名识别** — 将注册局保留域名与已注册、可注册域名区分开
- **多语言** — English（默认）、中文、日本語、한국어、Español
- **Redis 缓存** — 可选；未配置时会自动降级为无缓存模式
- **请求限流** — 基于 IP 的 token bucket，防止滥用
- **单二进制部署** — 所有静态资源通过 `go:embed` 内嵌

## 快速开始

### 前置要求

- Go 1.21+
- （可选）Node.js 18+，用于刷新注册商价格参考数据
- （可选）Redis，用于缓存

### 本地运行

```bash
go run .
```

服务默认启动在 `http://localhost:3300`。

### 构建

```bash
go build -o dmcheck .
./dmcheck
```

## 配置

### 配置文件

| 文件 | 说明 |
| ---- | ---- |
| `config/whois-servers.json` | TLD → WHOIS 服务器映射 |
| `config/default-tlds.json` | 展示给用户的默认 TLD 列表 |
| `config/registrar-prices.json` | 可选注册商链接模板和已配置的 TLD 价格参考；仅在 `REGISTRAR_PRICES_ENABLED=true` 时使用 |

### 更新注册商价格参考

注册商跳转和价格比价是可选功能，默认关闭。需要启用该功能的部署环境应设置 `REGISTRAR_PRICES_ENABLED=true`；如果不设置，新开发者或轻量部署不需要准备注册商价格数据，API 响应也不会返回 `registration_options`。

启用后，`config/registrar-prices.json` 里有两类相关但不同的数据：一类是面向用户展示的注册商跳转/搜索渠道，另一类是用于比价 UI 的自动价格行。某个注册商可以只作为跳转渠道存在，即使它暂时不进入自动价格比价。

当前覆盖情况以 `config/registrar-prices.json` 中的 `updated_at=2026-05-15` 为准：已启用 5 个注册商渠道；价格表总计覆盖 833 个后缀，其中包含 204 个多段后缀。

| 注册商 | 用户可见渠道 | 自动价格行数 | 更新方式 | 说明 |
| ------ | ------------ | ------------ | -------- | ---- |
| Cloudflare Registrar | 注册控制台跳转 | 0 | 仅跳转 | Cloudflare 发布了 registrar 定位和支持 TLD 信息，但没有适合无人值守刷新的大范围公开价格表。 |
| Porkbun | 注册/搜索跳转 + 比价 | 632 | 从官方公开价格页自动更新 | 用于提供较广的 TLD 价格覆盖。 |
| Namecheap | 注册/搜索跳转 | 0 | 仅跳转 | 公开价格页阻止脚本刷新，官方价格 API 需要账号凭证。 |
| Spaceship | 注册/搜索跳转 | 0 | 仅跳转 | `https://www.spaceship.com/domains/` 确实有公开价格页，并通过“See more”交互展示更多行；但无人值守更新脚本用 Node、curl、headless Chrome 访问时都会进入 Cloudflare verification。官方 API 也需要 key/secret 凭证，所以在没有稳定非交互数据源前保持为仅跳转。 |
| Dynadot | 注册/搜索跳转 + 比价 | 810 | 从官方公开价格页自动更新 | 用于提供较广的 TLD 价格覆盖，也包含较多多段后缀。 |

NameSilo 和 GoDaddy 已经过评估，但当前配置中还没有启用为渠道。只有当我们拿到稳定的无人值守公开数据源，或明确要做带凭证的集成时，才会加入。

```bash
node scripts/update-registrar-prices.mjs --date=YYYY-MM-DD
```

更新机制：

- 从仓库根目录运行脚本，建议每周刷新一次，并在每次涉及注册商行为的发布前刷新。
- 脚本会抓取 Porkbun 和 Dynadot 的官方公开价格页，解析首年注册价和续费价，清理非自动源的旧价格行，排序后写回 `config/registrar-prices.json`。
- 如果 Porkbun 返回少于 100 行，或 Dynadot 返回少于 300 行，脚本会拒绝覆盖配置，避免把反爬页面或临时错误页误写进价格表。
- 每次刷新后检查脚本打印的覆盖数量；只有当两个自动源都通过行数护栏时，才提交生成的 `config/registrar-prices.json`。

如果某个来源持续失败，就先不要把它放进自动价格源，直到它重新提供稳定的公开数据。

暂不自动抓取或展示价格的服务商：Namecheap、NameSilo、Spaceship、GoDaddy 需要账号/API 凭证、浏览器交互，或阻止脚本访问公开价格页；Cloudflare 发布了 registrar 定位和支持 TLD 信息，但没有适合无人值守刷新的大范围公开价格表。

### 环境变量

| 变量 | 默认值 | 说明 |
| ---- | ------ | ---- |
| `PORT` | `3300` | HTTP 服务端口 |
| `REDIS_ADDR` | `localhost:6379` | Redis 地址；设为空可禁用 Redis |
| `RATE_LIMIT` | `2` | 每个 IP 每秒允许的请求数 |
| `RATE_BURST` | `5` | 限流器突发容量 |
| `AVAILABLE_CACHE_TTL` | `0` | 可注册域名结果缓存时间；`0` 表示禁用可注册结果缓存 |
| `REGISTERED_CACHE_TTL` | `2160h` | 已注册/保留域名的最大缓存时间；到期前会刷新已注册域名 |
| `CACHE_TTL` | （空） | `AVAILABLE_CACHE_TTL` 的旧版别名 |
| `REGISTRAR_PRICES_ENABLED` | `false` | 从 `config/registrar-prices.json` 启用注册商跳转和价格比价 |
| `GA_ID` | （空） | Google Analytics Measurement ID；留空表示禁用 |

## 项目结构

```text
├── main.go              # 入口、路由和中间件
├── whois.go             # WHOIS/RDAP 查询逻辑
├── handlers.go          # HTTP handlers（搜索、WHOIS API）
├── ratelimit.go         # 基于 IP 的限流器
├── config.go            # 配置加载
├── config/
│   ├── whois-servers.json
│   ├── default-tlds.json
│   └── registrar-prices.json
├── scripts/
│   └── update-registrar-prices.mjs
├── static/
│   ├── index.html       # 主页面
│   ├── app.js           # 前端逻辑 + i18n
│   ├── style.css
│   ├── lang/            # 翻译文件
│   │   ├── en.json
│   │   ├── zh.json
│   │   ├── ja.json
│   │   ├── ko.json
│   │   └── es.json
│   ├── tos.html
│   ├── privacy.html
│   └── legal.css
└── deploy/              # systemd 和 nginx 配置
```

## 部署

### 1. VPS 前置要求

- Ubuntu 20.04+，并已安装 Nginx
- 域名 DNS A 记录指向 VPS IP
- （可选）Redis，用于缓存

### 2. 首次服务器初始化

```bash
# 创建应用目录
sudo mkdir -p /opt/dmcheck
sudo chown deployer:deployer /opt/dmcheck

# 本地构建并上传，或交给 GitHub Actions
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dmcheck .
scp dmcheck your-user@your-vps:/opt/dmcheck/

# 安装 systemd 服务
sudo cp deploy/dmcheck.service /etc/systemd/system/dmcheck.service
sudo systemctl daemon-reload
sudo systemctl enable dmcheck
sudo systemctl start dmcheck
```

### 3. Nginx + HTTPS

```bash
# 复制 HTTP-only 配置并启用
sudo cp deploy/dmcheck.nginx /etc/nginx/sites-available/dmcheck
sudo ln -s /etc/nginx/sites-available/dmcheck /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx

# 签发证书，certbot 会自动向 Nginx 写入 SSL 配置
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d dmcheck.app
```

Certbot 会修改 Nginx 配置，添加 SSL 监听、HTTP→HTTPS 跳转，并配置自动续期。

### 4. GitHub Actions 自动部署

`.github/workflows/deploy.yml` 会在每次推送到 `publish` 分支时自动构建并部署。

在 GitHub 仓库中添加以下 secrets（Settings → Secrets → Actions）：

| Secret | Value |
| ------ | ----- |
| `VPS_HOST` | 你的 VPS IP 或主机名 |
| `VPS_PORT` | SSH 端口；如果不是 22 才需要改 |
| `VPS_USER` | `deployer` |
| `VPS_SSH_KEY` | SSH 私钥 |

流程：`git push origin publish` → GitHub Actions 构建 `linux/amd64` 二进制 → `scp` 到 VPS → `systemctl restart dmcheck`。

### 5. VPS 环境变量

编辑 systemd service，添加环境变量：

```bash
sudo systemctl edit dmcheck
```

```ini
[Service]
Environment=RATE_LIMIT=5
Environment=RATE_BURST=10
Environment=REGISTERED_CACHE_TTL=2160h
# 可选：启用注册商跳转和价格比价
Environment=REGISTRAR_PRICES_ENABLED=true
Environment=GA_ID=G-XXXXXXXXXX
```

然后执行：

```bash
sudo systemctl restart dmcheck
```

## 致谢

- [IANA](https://data.iana.org/rdap/dns.json) — RDAP bootstrap 数据
- [rdap.org](https://rdap.org) — RDAP 查询 fallback
- [screenshot.domains](https://screenshot.domains) — 网站截图
- [favicon.im](https://favicon.im) — 网站 favicon
- [Porkbun domain pricing](https://porkbun.com/products/domains/) 和 [Dynadot domain pricing](https://www.dynadot.com/domain/prices) — 注册价格参考来源

## 贡献

关于添加翻译、WHOIS 服务器和提交 pull request 的说明，请查看 [CONTRIBUTING.md](CONTRIBUTING.md)。

## License

[AGPL-3.0](LICENSE)
