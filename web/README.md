# Nexus Scrims — Web

A modern web platform for Nexus Scrims, a competitive Apex Legends league and scrim server. Built with Angular 18, it provides player statistics, match history, league standings, and a custom ELO-based rating system.

---

## Key Features

- **Battle Royale Rating System** — multi-factor ELO tailored for 20-team BR games (placement, combat, damage, support, opponent strength, with a consistency bonus and BR-specific K-factor).
- **Interactive Rating Configurator** — live tool for admins to experiment with rating weights, K-factor, and normalization.
- **Component modularisation** — all major features are split into focused, reusable Angular standalone components.
- **Base Grid system** — a single configurable data-grid component (sorting, selection, responsive, custom cell templates).
- **Comprehensive player & match stats** — kills, deaths, damage, revives, respawns, and per-match breakdowns.
- **Live match tracking** — progress bars and in-game events for active matches.
- **ELO scrim leaderboards** — dynamic leaderboards with per-player analytics.
- **Modern UI** — dark theme, glass morphism, CSS variables, smooth animations.

---

## Project structure

```
web/
├── public/                     # Static assets (favicon, logo, banner)
├── scripts/                    # Build-time helpers
├── server/                     # Node backend (Express) + bundled scrim JSON
│   ├── index.ts                # /leaderboard, /league/* endpoints
│   └── scrims_batch/           # Historical scrim archive (static JSON)
├── src/
│   ├── app/
│   │   ├── components/         # Reusable UI components
│   │   ├── pages/              # Route-based page components
│   │   ├── services/           # Data loaders (incl. match-loader.service)
│   │   ├── models/             # Shared types
│   │   ├── app.routes.ts
│   │   └── app.config.ts
│   ├── styles.css              # Global Nexus brand styling
│   └── index.html
├── angular.json
├── proxy.conf.mjs              # Dev-server proxy (HF + local backend)
├── package.json
└── tsconfig.json
```

---

## Data sources

The app pulls data from two places, both routed through the dev-server proxy ([`proxy.conf.mjs`](./proxy.conf.mjs)):

| Path prefix | Target | Purpose |
|---|---|---|
| `/leaderboard`, `/league/*` | local Node backend (`server/index.ts`, default `http://localhost:3001`) | aggregated ELO + league endpoints backed by the bundled `server/scrims_batch/` archive |
| `/hf-api`, `/hf-resolve` | `huggingface.co/datasets/<HF_ORG>/<HF_SCRIMS_DATASET>` | live scrim batch files |
| `/hf-league-api`, `/hf-league-resolve` | `huggingface.co/datasets/<HF_ORG>/<HF_LEAGUE_DATASET>` | live league match + summary files |

### Configuring HuggingFace datasets

The HF org defaults to `Nexus-apex`. **That account does not yet exist publicly** — before running the app you must either:

1. Create a HuggingFace org / user with the expected dataset layout and set `HF_ORG` (and optionally `HF_SCRIMS_DATASET` / `HF_LEAGUE_DATASET`) when starting the dev server, or
2. Point `HF_ORG` at an existing dataset that follows the same schema.

```bash
# Windows PowerShell
$env:HF_ORG = "your-hf-org"; npm start

# Linux / macOS
HF_ORG=your-hf-org npm start
```

Expected dataset layout:

```
<HF_ORG>/<HF_SCRIMS_DATASET>           # default: Nexus-apex/apex-scrims
  └── scrim_YYYY_MM_DD_id_*.json

<HF_ORG>/<HF_LEAGUE_DATASET>           # default: Nexus-apex/apex-league
  └── Season_N/
        └── Division_N/
              ├── Week_*.json
              └── _summary.json
```

If you don't need live HF data, the bundled `server/scrims_batch/` archive is enough for local development — only the HF-backed pages will be empty.

---

## Quickstart

```bash
npm install

# 1. Start the local backend (leaderboard + league endpoints)
#    (see server/README.md for details)
npm run server    # if a server script is defined, otherwise: ts-node server/index.ts

# 2. In a second terminal, start the Angular dev server
npm start         # http://localhost:4200
```

---

## Scripts

| Script | Purpose |
|---|---|
| `npm start` | Angular dev server (`ng serve`) |
| `npm run build` | Production build to `dist/nexus-scrims-web/` |
| `npm run watch` | Dev build in watch mode |
| `npm test` | Karma/Jasmine unit tests |
| `npm run generate-summaries` | Regenerate `_summary.json` files from the league archive |

---

## Battle-royale rating system

Custom ELO-based rating designed for 20-team Apex lobbies:

- **Multi-factor evaluation** — placement (40%), combat (25%), damage (15%), support (10%), opponent strength (10%)
- **Consistency bonus** — rewards balanced performance across all factors
- **BR-specific K-factor** — higher volatility (K = 38.4) for faster adjustment
- **Continuous results** — non-binary, granular rating changes
- **Team and player tracking** — ratings stored for both

See [`BATTLE_ROYALE_RATING_SYSTEM.md`](./BATTLE_ROYALE_RATING_SYSTEM.md) and [`RATING_CONFIGURATOR_DOCUMENTATION.md`](./RATING_CONFIGURATOR_DOCUMENTATION.md) for the full spec.

---

## Technology stack

- Angular 18 (standalone components)
- TypeScript 5.5
- Node backend (Express, in `server/`)
- Karma / Jasmine for unit tests

---

## Contributing

1. Follow the Angular style guide.
2. Use the base grid for all data tables.
3. Maintain responsive design and the dark theme.
4. Keep Nexus Scrims branding consistent across code, UI copy, and docs.
5. Run `npm test` and `npm run build` before opening a PR.
