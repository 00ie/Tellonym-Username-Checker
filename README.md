<div align="center">

# TELLONYM USERNAME CHECKER

<br/>

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://golang.org/)
[![Wails](https://img.shields.io/badge/Wails-v2-FF3E00?style=for-the-badge)](https://wails.io/)
[![React](https://img.shields.io/badge/React-18-61DAFB?style=for-the-badge&logo=react&logoColor=black)](https://react.dev/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?style=for-the-badge&logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![TailwindCSS](https://img.shields.io/badge/TailwindCSS-3-06B6D4?style=for-the-badge&logo=tailwindcss&logoColor=white)](https://tailwindcss.com/)
[![License](https://img.shields.io/badge/License-MIT-22c55e?style=for-the-badge)](LICENSE)
[![Discord](https://img.shields.io/badge/Webhook-Discord-5865F2?style=for-the-badge&logo=discord&logoColor=white)](https://discord.com/)


</div>


---

## Features

| Feature | Description |
|---|---|
| **Checker Controls** | Start, pause, resume, and stop at any time no restart required |
| **Username Policy Engine** | Backend and frontend enforce Tellonym's exact validation rules |
| **Proxy Manager** | Bulk import, single test, batch validation, dead proxy cleanup, persistent storage |
| **Live Dashboard** | Real-time counters, trend charts, and rate-limit warning modal |
| **Historical Statistics** | Range presets, custom date filters, and granularity control |
| **Discord Webhook Alerts** | Instant notifications with username, timestamp, and direct profile link |
| **Theme System** | Red, blue, green, or purple choice persisted locally |
| **Multilingual UI** | Portuguese and English |

---

## Tech Stack

| Layer | Technology | Version |
|---|---|---|
| Backend | Go | 1.21+ |
| Desktop Bridge | Wails | v2 |
| Frontend | React + TypeScript | 18 / 5 |
| State Management | Zustand | latest |
| Styling | TailwindCSS | 3 |
| Charts | Chart.js | latest |

---

## Requirements

- **Go** 1.21 or newer
- **Node.js** 18 or newer + **npm**
- **Wails CLI**
- **WebView2 Runtime** *(Windows only usually pre-installed on Windows 11)*

**Install Wails CLI:**

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

**Validate your environment:**

```bash
wails doctor
```


---

## Installation and Execution

### 1. Clone the repository

```bash
git clone https://github.com/00ie/Tellonym-Username-Checker.git
cd "Tellonym-Username-Checker"
```

### 2. Install dependencies

```bash
go mod tidy
cd frontend
npm install
cd ..
```

### 3. Run in development mode

```bash
wails dev
```

### 4. Build for production

```bash
wails build
```

### 5. Run the generated executable

Windows output path:

```bash
build/bin/TellonymUsernameChecker.exe
```

---

## Username Rules

The engine validates every username against Tellonym's policy before sending a request:

| Rule | Detail |
|---|---|
| Allowed characters | Letters `a–z` · Numbers `0–9` · Underscore `_` · Dot `.` |
| Minimum length | `3` characters |
| Maximum length | `30` characters |
| Blocked: leading dot | `.username` → rejected |
| Blocked: trailing dot | `username.` → rejected |
| Blocked: consecutive dots | `user..name` → rejected |

Profile URL format: `https://tellonym.me/<username>`

---

## Configuration

| Path | Purpose |
|---|---|
| `configs/config.yaml` | Base config committed to the repository |
| `tellonym checker/config/config.yaml` | Runtime config machine-specific, not committed |

<details>
<summary><strong>View all configuration fields</strong></summary>

<br/>

| Field | Description |
|---|---|
| `checker.request_timeout` | HTTP request timeout per check |
| `checker.max_retries` | Max retry attempts per username |
| `checker.batch_size` | Usernames processed per batch |
| `checker.queue_size` | Internal queue capacity |
| `checker.max_concurrent` | Maximum concurrent workers |
| `checker.username_rules` | Username validation policy |
| `proxy.types` | Allowed proxy protocol types |
| `proxy.max_consecutive_fails` | Consecutive failures before blacklisting a proxy |
| `proxy.health_check_interval` | Interval between automatic health checks |
| `proxy.validation_timeout` | Timeout for proxy validation requests |
| `webhook.enabled` | Toggle Discord notifications on/off |
| `webhook.url` | Your Discord webhook URL |
| `webhook.timeout` | Webhook HTTP request timeout |

</details>

---

## Proxy System

**Accepted formats:**

```
host:port
http://host:port
https://host:port
socks4://host:port
socks5://host:port
http://user:pass@host:port
```

One proxy per line in `tellonym checker/proxies.txt`.

**Lifecycle:**

```
Load & Normalize  ──►  Validate on Startup  ──►  Continuous Health Checks
       │                                                     │
       ▼                                                     ▼
  Build Pool                                      Remove Dead Proxies
       │                                                     │
       ▼                                                     ▼
  Rotate on Use  ◄────────────────────────────  Persist to proxies.txt
```

---

## Webhook Notifications

When an available username is found, a Discord embed is sent with:

- The available username
- Timestamp of discovery
- Direct link: `https://tellonym.me/<username>`
- Fixed sender identity (configured in backend settings)

**Setup:**

```
1. Open Settings in the app
2. Toggle webhook → Enabled
3. Paste your Discord webhook URL
4. Save settings
```

---

## Dashboard & Statistics

**Live counters:**

```
┌─────────────┬────────────┬──────────┬────────┬──────────────┬──────────┐
│  Attempts   │   Found    │  Errors  │  Rate  │ Avg Response │  Uptime  │
└─────────────┴────────────┴──────────┴────────┴──────────────┴──────────┘
```

**Statistics filters:**

| Filter | Options |
|---|---|
| Time range | `Last 1h` · `Last 24h` · `Last 7d` · `Last 30d` · `Today` · `Custom` |
| Granularity | `Minute` · `Hour` · `Day` · `Auto` |
| Export | `CSV` · `JSON` |

---

## Runtime Folder Layout

Created automatically on first launch:

```
tellonym checker/
├── config/
│   └── config.yaml            ← effective runtime configuration
├── data/
│   └── found_usernames.txt    ← all discovered available usernames
├── logs/
│   └── app.log                ← application log
├── exports/                   ← manual CSV / JSON exports
└── proxies.txt                ← active proxy pool
```

> Runtime files are machine-specific. They are excluded via `.gitignore` and must never be committed.

---

## Project Structure

```
.
├── main.go
├── wails.json
├── configs/
├── build/
│
├── backend/
│   ├── app.go
│   ├── runtime_layout.go
│   ├── api/
│   ├── core/
│   │   ├── checker/           ← worker pool, check logic
│   │   ├── proxy/             ← proxy manager, health checks
│   │   ├── storage/           ← persistence layer
│   │   ├── config/            ← config loader
│   │   └── models/            ← shared data models
│   ├── services/
│   └── utils/
│
└── frontend/
    └── src/
        ├── components/        ← UI components
        ├── hooks/             ← custom React hooks
        ├── i18n/              ← translations (PT / EN)
        ├── services/          ← Wails API bindings
        ├── store/             ← Zustand state
        ├── theme/             ← theme configuration
        ├── types/             ← TypeScript types
        └── utils/
```

---

## Quality & Validation

```bash
# Backend — unit tests
go test ./...

# Backend — static analysis
go vet ./...

# Frontend — production build
cd frontend && npm run build

# Wails — compile check (no binary output)
wails build -s -skipbindings -o TellonymUsernameChecker-check
```

---

## Troubleshooting

<details>
<summary><strong>High error rate or low throughput</strong></summary>
<br/>

1. Open **Proxy Manager** and run batch validation
2. Remove all dead proxies
3. Reduce thread count in Settings
4. Increase `request_timeout` and `max_retries` in config
5. Keep healthy proxy count well above active thread count

</details>

<details>
<summary><strong><code>no proxies loaded</code> in logs</strong></summary>
<br/>

`tellonym checker/proxies.txt` is empty or missing.

1. Add valid proxies, one per line
2. Use the Proxy Manager to test and clean the pool
3. Restart the checker

</details>

<details>
<summary><strong>Theme appears inconsistent</strong></summary>
<br/>

1. Re-select the theme in **Settings**
2. Restart the app
3. If the issue persists: delete `frontend/dist` and run `npm run build`

</details>

<details>
<summary><strong>Webhook 429 — rate limited</strong></summary>
<br/>

1. Reduce concurrent threads
2. Rotate to higher-quality proxies
3. Disable webhook temporarily while rate-limited
4. Wait for the rate-limit window to reset

</details>

---
