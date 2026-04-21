// Package models defines shared types aligned with the scrim-bot nhost schema.
package models

import "time"

// Player mirrors the nhost `players` table.
type Player struct {
	ID          string  `json:"id"`
	DiscordID   string  `json:"discord_id"`
	DisplayName string  `json:"display_name"`
	OverstatID  string  `json:"overstat_id,omitempty"`
	Elo         float64 `json:"elo,omitempty"`
}

// Team is a group of 3 players that signed up together.
type Team struct {
	// TeamName from scrim_signups.team_name
	TeamName string `json:"team_name"`
	// SignupID is the scrim_signups row id (if known)
	SignupID string `json:"signup_id,omitempty"`
	// ScrimID links back to the scrim
	ScrimID string `json:"scrim_id"`
	// SignupPlayer is the user who ran the signup command
	SignupPlayer Player `json:"signup_player"`
	// Players are the 3 rostered players
	Players [3]Player `json:"players"`
	// CombinedElo is the pre-computed elo sum stored in nhost (may be 0)
	CombinedElo float64 `json:"combined_elo,omitempty"`
	// Prio amount (higher = more priority)
	Prio float64 `json:"prio,omitempty"`
	// SignupTime is when the team signed up
	SignupTime time.Time `json:"signup_time"`
}

// QueueEntry wraps a Team with matchmaking metadata.
type QueueEntry struct {
	Team       Team      `json:"team"`
	MMR        float64   `json:"mmr"`        // team MMR from adapter
	EnqueuedAt time.Time `json:"enqueued_at"`
}

// Lobby is a group of teams assigned to play together.
type Lobby struct {
	ID        string       `json:"id"`
	ScrimID   string       `json:"scrim_id"`
	Teams     []QueueEntry `json:"teams"`
	AvgMMR    float64      `json:"avg_mmr"`
	MMRSpread float64      `json:"mmr_spread"`
	CreatedAt time.Time    `json:"created_at"`
}

// MatchResult is posted after a game ends.
type MatchResult struct {
	LobbyID string            `json:"lobby_id"`
	Teams   []TeamPlacement   `json:"teams"`
}

// TeamPlacement records a team's outcome.
type TeamPlacement struct {
	TeamName  string `json:"team_name"`
	Placement int    `json:"placement"`
	Score     int    `json:"score"`
}
