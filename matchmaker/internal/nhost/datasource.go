package nhost

import (
	"context"

	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/models"
)

// DataSource is the interface the API server uses to fetch data.
// In production this is *Client (real nhost). In dev mode it's *Stub.
type DataSource interface {
	GetSignupsForScrim(ctx context.Context, scrimID string) ([]models.Team, error)
	GetActiveScrims(ctx context.Context) ([]ActiveScrim, error)
}
