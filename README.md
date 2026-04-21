# Nexus Scrims — Apex Legends Scrim Platform

A full-stack platform for organising competitive Apex Legends scrims and leagues under the **Nexus Scrims** brand. This monorepo bundles three services:

| Folder | Stack | Purpose |
|---|---|---|
| [`web/`](./web) | Angular 18 + Node SSR | Public website — home, league standings, scrim archive, player stats, ELO/rating configurator |
| [`matchmaker/`](./matchmaker) | Go (chi router) | Lightweight matchmaking service that groups signed-up teams into balanced lobbies |
| [`bot/`](./bot) | TypeScript + discord.js | Discord bot for running scrim signups, rosters, and match coordination |

## Architecture

```
                   ┌──────────────────┐
                   │   Discord users  │
                   └────────┬─────────┘
                            │
                   ┌────────▼─────────┐           ┌───────────────┐
                   │     bot/         │  signup   │  matchmaker/  │
                   │  (discord.js)    ├──────────►│   (Go API)    │
                   └────────┬─────────┘  webhook  └──────┬────────┘
                            │                            │
                            │ writes scrim data          │ produces lobbies
                            ▼                            ▼
                   ┌──────────────────────────────────────────┐
                   │        nhost Postgres (shared)           │
                   └──────────────────┬───────────────────────┘
                                      │
                                      ▼
                             ┌──────────────────┐
                             │      web/        │
                             │   (Angular 18)   │
                             └──────────────────┘
```

## Quickstart

The repo is set up as a small npm workspace (`web` + `bot`) plus a Go module (`matchmaker`). From the root:

```bash
# One-time setup: installs JS deps for web + bot and tidies Go deps for matchmaker.
npm run install:all

# Run each service in its own terminal:
npm run dev:web          # Angular dev server on http://localhost:4200
npm run dev:matchmaker   # Go matchmaker on http://localhost:8080
npm run dev:bot          # Discord bot (requires bot/config.json)
```

See per-service READMEs for details:

- [`web/README.md`](./web/README.md) — feature list, scripts, HuggingFace configuration
- [`matchmaker/README.md`](./matchmaker/README.md) — API reference, payloads, deployment
- [`bot/README.md`](./bot/README.md) — setup, commands, token handling

### Run everything at once

```bash
npm run build         # build all three services
npm test              # run all test suites (web Karma, bot Jest, Go)
```

## Prerequisites

| Tool | Version |
|---|---|
| Node.js | ≥ 20 |
| Go | ≥ 1.22 |
| (optional) Docker | for containerised matchmaker |
| (optional) nhost | for real data; `DEV_MODE=true` avoids it for the matchmaker |

## Configuration summary

| Service | Secret / config | Where |
|---|---|---|
| `bot` | Discord bot token, client id, guild id | `bot/config.json` (copy from `bot/config.example.json`, never commit) |
| `matchmaker` | `WEBHOOK_SECRET`, `NHOST_*`, `LOBBY_SIZE` | env vars (see `matchmaker/.env.example`) |
| `web` | `HF_ORG`, `HF_SCRIMS_DATASET`, `HF_LEAGUE_DATASET`, `BACKEND_URL` | env vars passed to `npm start` (see `web/proxy.conf.mjs` + web README) |

> **Heads-up:** the web app's HuggingFace proxy defaults point at an `Nexus-apex` org that must be created or populated with the expected dataset layout. See [`web/README.md`](./web/README.md#configuring-huggingface-datasets) for details and alternatives.

## Repository layout

```
apex-legends-scrim-bot/
├── README.md
├── LICENSE                       # MIT
├── package.json                  # root workspace: web + bot
├── .gitignore
├── .github/workflows/ci.yml      # build + test all three services on PR
├── web/                          # Angular frontend + SSR server
├── matchmaker/                   # Go matchmaking microservice
└── bot/                          # Discord scrim bot (TypeScript)
```

## CI

Every push and pull request runs three jobs:

- **web** — `npm ci && npm run build`
- **bot** — `npm ci && npm run build && npm test`
- **matchmaker** — `go build ./... && go test ./...`

See [`.github/workflows/ci.yml`](./.github/workflows/ci.yml).

## License

MIT. See [LICENSE](./LICENSE).
