# Nexus Matchmaker

Lightweight matchmaking service for Apex Legends custom games (Nexus Scrims).
Teams of 3 sign up via the Discord bot (see [`../bot`](../bot)), then this
service groups them into balanced lobbies.

## Architecture

```
Discord Bot (../bot)
  │  writes signups to nhost Postgres
  │  POST /webhook/signup ───────────────┐
  │                                      ▼
  │                              ┌──────────────┐
  │                              │  Matchmaker   │
  │                              │  (this repo)  │
  │                              │               │
  │  ◄── POST /notify/match ─── │  Queue → Batch│──► MMR Adapter (pluggable)
  │                              │  → Lobby      │
  │                              └──────────────┘
  │                                      │
  └──────────────────────────────────────┘
```

## Project layout

```
cmd/server/main.go                — entrypoint
internal/
  api/
    server.go                     — HTTP handlers + HMAC middleware (chi router)
    server_test.go                — handler unit tests (httptest)
  config/config.go                — env-based configuration
  matchmaker/
    matchmaker.go                 — Queue + Batcher (lobby balancer)
    matchmaker_test.go            — batcher / queue unit tests
  mmr/
    adapter.go                    — MMR adapter interface
    mock.go                       — mock adapter (avg elo / random)
  models/models.go                — shared types (Player, Team, Lobby, etc.)
  nhost/
    datasource.go                 — DataSource interface
    client.go                     — real GraphQL client for nhost Postgres
    stub.go                       — dev-mode stub with fake generated data
endpoints.md                      — full API & payload spec
Dockerfile                        — multi-stage Docker build
```

## Data model alignment with scrim-bot

The models mirror the nhost schema used by scrim-bot:

| Matchmaker type | nhost table / field |
|---|---|
| `Player.ID` | `players.id` (uuid) |
| `Player.DiscordID` | `players.discord_id` |
| `Player.Elo` | `players.elo` |
| `Team.TeamName` | `scrim_signups.team_name` |
| `Team.ScrimID` | `scrim_signups.scrim_id` |
| `Team.Players[0..2]` | `player_one_id`, `player_two_id`, `player_three_id` |
| `Team.CombinedElo` | `scrim_signups.combined_elo` |

## Quickstart

### Prerequisites

- Go 1.22+
- (Optional) Docker for containerised deployment
- nhost running locally or remotely

### Run locally (dev mode)

Dev mode starts the server without touching nhost or the Discord bot.
All signups return fake generated teams and HMAC verification is bypassed.

```bash
go mod tidy

# Windows PowerShell
$env:DEV_MODE="true"; $env:LOBBY_SIZE="3"; go run ./cmd/server

# Linux / macOS
DEV_MODE=true LOBBY_SIZE=3 go run ./cmd/server
```

### Run locally (production)

```bash
go mod tidy

export LOBBY_SIZE=20
export NHOST_GRAPHQL_URL=http://localhost:1337/v1/graphql
export NHOST_ADMIN_SECRET=nhost-admin-secret
export WEBHOOK_SECRET=<your-hmac-secret>

go run ./cmd/server
```

### Run tests

```bash
go test ./...
```

All tests run in dev mode using the stub `DataSource` — no external calls are made.

### Docker

```bash
docker build -t nexus-matchmaker .
docker run -p 8080:8080 \
  -e LOBBY_SIZE=20 \
  -e NHOST_GRAPHQL_URL=http://host.docker.internal:1337/v1/graphql \
  -e NHOST_ADMIN_SECRET=nhost-admin-secret \
  nexus-matchmaker
```

## API endpoints

| Method | Path | Auth | Purpose |
|---|---|---|---|
| GET | `/health` | none | Health check |
| POST | `/webhook/signup` | HMAC | Bot sends team signup |
| POST | `/webhook/cancel` | HMAC | Bot cancels a team |
| POST | `/webhook/match_result` | HMAC | Post match results for MMR |
| GET | `/status/queue` | none | Queue depth |
| GET | `/lobbies` | none | List formed lobbies |
| POST | `/admin/enqueue_scrim/{scrimID}` | none | Pull signups from nhost & enqueue |
| POST | `/admin/batch` | none | Force one batch cycle |

See [endpoints.md](endpoints.md) for full payload examples.

## How matching works

1. Teams are enqueued (via webhook or admin pull from nhost).
2. The MMR adapter assigns each team an MMR (mock returns avg elo or random).
3. The batcher runs every 15s (configurable) or on-demand via `/admin/batch`.
4. Teams are sorted by MMR, then grouped into consecutive lobbies of `LOBBY_SIZE`.
5. Leftover teams stay in the queue for the next cycle.

## MMR adapter

The `mmr.Adapter` interface (`internal/mmr/adapter.go`) has three methods:

```go
GetPlayerMMR(ctx, playerID) (float64, error)
GetTeamMMR(ctx, players)    (float64, error)
UpdateResults(ctx, result)  error
```

Currently `mmr.Mock` is wired in. Implement a new struct satisfying the interface
to plug in any MMR service.

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `ADDR` | `:8080` | HTTP listen address |
| `LOBBY_SIZE` | `20` | Teams per lobby (use `3` for quick dev testing) |
| `WEBHOOK_SECRET` | `dev-secret` | HMAC key for webhook auth. **Required in production.** Requests to `/webhook/*` are rejected with `503` when the secret is empty or left at the `dev-secret` default, unless `DEV_MODE=true`. |
| `NHOST_GRAPHQL_URL` | `http://localhost:1337/v1/graphql` | Hasura GraphQL endpoint |
| `NHOST_ADMIN_SECRET` | `nhost-admin-secret` | Hasura admin secret |
| `DEV_MODE` | `false` | Set `true` to disable all external calls and use stub data |

## Next steps

- [ ] Wire bot to call `/webhook/signup` on team signups
- [ ] Implement real MMR adapter
- [ ] Add lobby-ready notification callback to bot
- [ ] Persistence layer for lobby audit trail
- [ ] Metrics (Prometheus) and structured logging

