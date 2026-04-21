package nhost

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/models"
)

// Stub is a dev-mode DataSource that returns generated fake teams.
// It never contacts nhost or any external service.
type Stub struct {
	logger    *slog.Logger
	teamCount int // how many fake teams to generate per scrim
}

func NewStub(logger *slog.Logger, teamCount int) *Stub {
	if teamCount <= 0 {
		teamCount = 20
	}
	return &Stub{logger: logger, teamCount: teamCount}
}

func (s *Stub) GetSignupsForScrim(_ context.Context, scrimID string) ([]models.Team, error) {
	s.logger.Info("[DEV] generating fake signups", "scrim_id", scrimID, "count", s.teamCount)
	return GenerateFakeTeams(scrimID, s.teamCount), nil
}

func (s *Stub) GetActiveScrims(_ context.Context) ([]ActiveScrim, error) {
	s.logger.Info("[DEV] returning fake active scrims")
	return []ActiveScrim{
		{ID: "dev-scrim-1", DiscordChannel: "dev-channel", DateTimeField: time.Now().UTC().Format(time.RFC3339)},
	}, nil
}

// ---------- fake data generator ----------

// GenerateFakeTeams creates n teams with random elo values for testing.
func GenerateFakeTeams(scrimID string, n int) []models.Team {
	teams := make([]models.Team, 0, n)
	for i := 1; i <= n; i++ {
		players := [3]models.Player{
			fakePlayer(i, 1),
			fakePlayer(i, 2),
			fakePlayer(i, 3),
		}
		teams = append(teams, models.Team{
			TeamName: fmt.Sprintf("Team_%d", i),
			ScrimID:  scrimID,
			SignupPlayer: models.Player{
				ID:          fmt.Sprintf("signup_%d", i),
				DiscordID:   fmt.Sprintf("d_signup_%d", i),
				DisplayName: fmt.Sprintf("Captain%d", i),
			},
			Players:     players,
			CombinedElo: players[0].Elo + players[1].Elo + players[2].Elo,
			SignupTime:  time.Now().UTC().Add(-time.Duration(n-i) * time.Minute),
		})
	}
	return teams
}

func fakePlayer(teamIdx, slot int) models.Player {
	elo := 1000 + rand.Float64()*2000 // 1000–3000
	return models.Player{
		ID:          fmt.Sprintf("p_%d_%d", teamIdx, slot),
		DiscordID:   fmt.Sprintf("d_%d_%d", teamIdx, slot),
		DisplayName: fmt.Sprintf("Player%d_%d", teamIdx, slot),
		Elo:         elo,
	}
}
