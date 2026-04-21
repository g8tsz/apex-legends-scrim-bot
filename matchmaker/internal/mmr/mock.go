package mmr

import (
	"context"
	"math/rand/v2"

	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/models"
)

// Mock is a placeholder adapter that returns the player's stored elo or a
// random value. Replace with a real provider when the MMR service is ready.
type Mock struct{}

func NewMock() *Mock { return &Mock{} }

func (m *Mock) GetPlayerMMR(_ context.Context, _ string) (float64, error) {
	// Return a random MMR between 1000-3000 when we have no real data.
	return 1000 + rand.Float64()*2000, nil
}

func (m *Mock) GetTeamMMR(_ context.Context, players [3]models.Player) (float64, error) {
	var sum float64
	for _, p := range players {
		if p.Elo > 0 {
			sum += p.Elo
		} else {
			sum += 1500 // default placeholder
		}
	}
	return sum / 3, nil
}

func (m *Mock) UpdateResults(_ context.Context, _ models.MatchResult) error {
	return nil // no-op for mock
}
