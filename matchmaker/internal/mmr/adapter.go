// Package mmr defines the pluggable MMR adapter interface.
package mmr

import (
	"context"

	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/models"
)

// Adapter is the interface any MMR provider must implement.
// Swap the concrete implementation to switch MMR backends.
type Adapter interface {
	// GetPlayerMMR returns the MMR for a single player.
	GetPlayerMMR(ctx context.Context, playerID string) (float64, error)

	// GetTeamMMR returns a composite MMR for a team of players.
	GetTeamMMR(ctx context.Context, players [3]models.Player) (float64, error)

	// UpdateResults sends match results so the MMR provider can adjust ratings.
	UpdateResults(ctx context.Context, result models.MatchResult) error
}
