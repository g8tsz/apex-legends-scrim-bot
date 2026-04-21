# Matchmaker API & Integration Endpoints

This document specifies the HTTP endpoints, expected payloads, and integration points for the Discord bot and the MMR service adapter.

Guidelines
- All incoming webhooks must include `X-Matchmaker-Signature` HMAC header (shared secret `MATCHMAKER_WEBHOOK_SECRET`).
- Use JSON over HTTPS. Authenticate admin endpoints with an API token or mTLS.

Bot -> Matchmaker (incoming webhooks)

1) POST /webhook/signup
- Purpose: Bot notifies matchmaker of a new team signup (teams of 3).
- Payload example:

```json
{
  "team_id": "team_123",
  "players": [
    { "discord_id": "12345", "display_name": "PlayerA" },
    { "discord_id": "23456", "display_name": "PlayerB" },
    { "discord_id": "34567", "display_name": "PlayerC" }
  ],
  "region": "NA",
  "signup_time": "2026-04-19T12:34:56Z",
  "source": "discord"
}
```

Behavior: matchmaker enqueues team, fetches cached MMR (or uses mock), returns `202 Accepted` on success.

2) POST /webhook/cancel
- Purpose: Bot notifies that a team cancelled prior to matchmaking.
- Payload: `{ "team_id": "team_123" }`

3) POST /webhook/match_result
- Purpose: Bot or game server posts final match results for MMR updates and auditing.
- Payload example (simplified):

```json
{
  "match_id": "m_789",
  "lobby_id": "l_456",
  "teams": [
    { "team_id": "team_123", "placement": 1, "score": 20 },
    { "team_id": "team_234", "placement": 2, "score": 15 }
  ],
  "events": []
}
```

Matchmaker -> Bot (outgoing notifications)

1) POST /notify/match_start
- Purpose: Inform bot/lobby manager that a lobby has been created and teams are locked.
- Payload example:

```json
{
  "match_id": "m_789",
  "lobby_id": "l_456",
  "teams": ["team_123", "team_234"],
  "server": { "region": "NA", "ip": "1.2.3.4", "port": 7777 },
  "start_by": "matchmaker"
}
```

MMR Adapter Interface (pluggable)

The MMR adapter can be implemented as an internal module or an external service. The matchmaker expects these calls:

1) GET /mmr/player/:player_id
- Returns player's MMR and last-updated timestamp.

Response example:

```json
{ "player_id": "12345", "mmr": 2650, "mmr_updated_at": "2026-04-19T10:00:00Z" }
```

2) POST /mmr/team
- Given a list of player IDs, return team MMR summary.

Request example:

```json
{ "player_ids": ["12345","23456","34567"] }
```

Response example:

```json
{ "team_mmr_mean": 2600, "team_mmr_stddev": 45 }
```

3) POST /mmr/update_results
- Accepts match results to update player MMRs.

Request example:

```json
{
  "match_id": "m_789",
  "teams": [ { "team_id": "team_123", "players": ["12345","23456","34567"], "placement": 1 } ]
}
```

Admin / Observability Endpoints

- GET /status/queue — returns queue lengths per region and average wait times.
- GET /matches/:id — view match record and audit trail.
- POST /simulate — run a local simulation of the batcher (accepts parameters for team counts and MMR distribution).

Concurrency & Locking
- Matchmaker must acquire an advisory DB lock or Redis distributed lock before assigning teams into a match to prevent double assignment.

Retry and Patience
- Teams should have a configurable patience window. As wait time increases, the matchmaker relaxes MMR tolerance to reduce queue times.

Security
- All webhooks must be HMAC-signed. Admin endpoints require API key.

Notes for the Discord bot implementer
- Prefer sending signups directly to matchmaker `/webhook/signup` in addition to writing signup records to `nhost` Postgres — this reduces latency.
- Include `team_id` UUID that the matchmaker can reference in DB.

Notes for MMR provider implementer
- Support batched team MMR calculations for efficiency.
- Provide low-latency read endpoints for fetching cached MMRs.
