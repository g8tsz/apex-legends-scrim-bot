// Package nhost provides a lightweight GraphQL client for the nhost Postgres DB
// used by the scrim-bot.
package nhost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/models"
)

// Client talks to the nhost Hasura GraphQL endpoint.
type Client struct {
	url         string
	adminSecret string
	http        *http.Client
}

func NewClient(graphqlURL, adminSecret string) *Client {
	return &Client{
		url:         graphqlURL,
		adminSecret: adminSecret,
		http:        &http.Client{Timeout: 10 * time.Second},
	}
}

// graphqlRequest is a generic helper that sends a GraphQL query/mutation.
func (c *Client) graphqlRequest(ctx context.Context, query string, target any) error {
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return fmt.Errorf("marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", c.adminSecret)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("graphql status %d: %s", resp.StatusCode, string(respBody))
	}

	// Hasura wraps results in {"data": ...}
	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors json.RawMessage `json:"errors"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}
	if len(envelope.Errors) > 0 && string(envelope.Errors) != "null" {
		return fmt.Errorf("graphql errors: %s", string(envelope.Errors))
	}
	if target != nil {
		if err := json.Unmarshal(envelope.Data, target); err != nil {
			return fmt.Errorf("unmarshal data: %w", err)
		}
	}
	return nil
}

// ---------- read helpers ----------

// ScrimSignupRow mirrors the get_scrim_signups_with_players Hasura function output.
type ScrimSignupRow struct {
	ScrimID                  string  `json:"scrim_id"`
	DateTime                 string  `json:"date_time"`
	TeamName                 string  `json:"team_name"`
	SignupPlayerID           string  `json:"signup_player_id"`
	SignupPlayerDiscordID    string  `json:"signup_player_discord_id"`
	SignupPlayerDisplayName  string  `json:"signup_player_display_name"`
	PlayerOneID              string  `json:"player_one_id"`
	PlayerOneDiscordID       string  `json:"player_one_discord_id"`
	PlayerOneDisplayName     string  `json:"player_one_display_name"`
	PlayerOneOverstatID      string  `json:"player_one_overstat_id"`
	PlayerOneElo             float64 `json:"player_one_elo"`
	PlayerTwoID              string  `json:"player_two_id"`
	PlayerTwoDiscordID       string  `json:"player_two_discord_id"`
	PlayerTwoDisplayName     string  `json:"player_two_display_name"`
	PlayerTwoOverstatID      string  `json:"player_two_overstat_id"`
	PlayerTwoElo             float64 `json:"player_two_elo"`
	PlayerThreeID            string  `json:"player_three_id"`
	PlayerThreeDiscordID     string  `json:"player_three_discord_id"`
	PlayerThreeDisplayName   string  `json:"player_three_display_name"`
	PlayerThreeOverstatID    string  `json:"player_three_overstat_id"`
	PlayerThreeElo           float64 `json:"player_three_elo"`
}

// GetSignupsForScrim fetches all team signups for a given scrim using the same
// Hasura function the bot uses: get_scrim_signups_with_players.
func (c *Client) GetSignupsForScrim(ctx context.Context, scrimID string) ([]models.Team, error) {
	query := fmt.Sprintf(`query {
		get_scrim_signups_with_players(args: { scrim_id_search: "%s" }) {
			scrim_id date_time team_name
			signup_player_id signup_player_discord_id signup_player_display_name
			player_one_id player_one_discord_id player_one_display_name player_one_overstat_id player_one_elo
			player_two_id player_two_discord_id player_two_display_name player_two_overstat_id player_two_elo
			player_three_id player_three_discord_id player_three_display_name player_three_overstat_id player_three_elo
		}
	}`, scrimID)

	var result struct {
		Rows []ScrimSignupRow `json:"get_scrim_signups_with_players"`
	}
	if err := c.graphqlRequest(ctx, query, &result); err != nil {
		return nil, err
	}

	teams := make([]models.Team, 0, len(result.Rows))
	for _, r := range result.Rows {
		t, _ := time.Parse(time.RFC3339, r.DateTime)
		teams = append(teams, models.Team{
			TeamName: r.TeamName,
			ScrimID:  r.ScrimID,
			SignupPlayer: models.Player{
				ID:          r.SignupPlayerID,
				DiscordID:   r.SignupPlayerDiscordID,
				DisplayName: r.SignupPlayerDisplayName,
			},
			Players: [3]models.Player{
				{ID: r.PlayerOneID, DiscordID: r.PlayerOneDiscordID, DisplayName: r.PlayerOneDisplayName, OverstatID: r.PlayerOneOverstatID, Elo: r.PlayerOneElo},
				{ID: r.PlayerTwoID, DiscordID: r.PlayerTwoDiscordID, DisplayName: r.PlayerTwoDisplayName, OverstatID: r.PlayerTwoOverstatID, Elo: r.PlayerTwoElo},
				{ID: r.PlayerThreeID, DiscordID: r.PlayerThreeDiscordID, DisplayName: r.PlayerThreeDisplayName, OverstatID: r.PlayerThreeOverstatID, Elo: r.PlayerThreeElo},
			},
			CombinedElo: r.PlayerOneElo + r.PlayerTwoElo + r.PlayerThreeElo,
			SignupTime:   t,
		})
	}
	return teams, nil
}

// ActiveScrim mirrors the scrims table row.
type ActiveScrim struct {
	ID             string `json:"id"`
	DiscordChannel string `json:"discord_channel"`
	DateTimeField  string `json:"date_time_field"`
}

// GetActiveScrims returns all scrims where active=true.
func (c *Client) GetActiveScrims(ctx context.Context) ([]ActiveScrim, error) {
	query := `query {
		scrims(where: { active: { _eq: true } }) {
			id discord_channel date_time_field
		}
	}`
	var result struct {
		Scrims []ActiveScrim `json:"scrims"`
	}
	if err := c.graphqlRequest(ctx, query, &result); err != nil {
		return nil, err
	}
	return result.Scrims, nil
}
