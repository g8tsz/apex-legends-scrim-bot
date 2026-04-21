package matchmaker

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/mmr"
	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/models"
	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/nhost"
)

// ---------- Queue tests ----------

func TestQueue_EnqueueAndLen(t *testing.T) {
	q := NewQueue()
	if q.Len() != 0 {
		t.Fatalf("expected empty queue, got %d", q.Len())
	}

	q.Enqueue(makeEntry("team1", 1500))
	q.Enqueue(makeEntry("team2", 1600))

	if q.Len() != 2 {
		t.Fatalf("expected 2, got %d", q.Len())
	}
}

func TestQueue_Drain(t *testing.T) {
	q := NewQueue()
	q.Enqueue(makeEntry("a", 1000))
	q.Enqueue(makeEntry("b", 2000))

	entries := q.Drain()
	if len(entries) != 2 {
		t.Fatalf("expected 2 drained, got %d", len(entries))
	}
	if q.Len() != 0 {
		t.Fatalf("queue should be empty after drain, got %d", q.Len())
	}
}

func TestQueue_DrainEmpty(t *testing.T) {
	q := NewQueue()
	entries := q.Drain()
	if entries != nil {
		t.Fatalf("expected nil from draining empty queue, got %v", entries)
	}
}

// ---------- Batcher tests ----------

func TestBatcher_NotEnoughTeams(t *testing.T) {
	q := NewQueue()
	adapter := mmr.NewMock()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	b := NewBatcher(q, adapter, 3, time.Hour, logger) // lobbySize=3

	// Enqueue only 2 teams (less than lobbySize).
	q.Enqueue(makeEntry("t1", 1500))
	q.Enqueue(makeEntry("t2", 1600))

	lobbies := b.TriggerNow(context.Background())
	if len(lobbies) != 0 {
		t.Fatalf("expected 0 lobbies, got %d", len(lobbies))
	}
	// Teams should be re-enqueued.
	if q.Len() != 2 {
		t.Fatalf("expected 2 teams still in queue, got %d", q.Len())
	}
}

func TestBatcher_ExactLobby(t *testing.T) {
	q := NewQueue()
	adapter := mmr.NewMock()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	b := NewBatcher(q, adapter, 3, time.Hour, logger)

	q.Enqueue(makeEntry("low", 1000))
	q.Enqueue(makeEntry("mid", 1500))
	q.Enqueue(makeEntry("high", 2000))

	lobbies := b.TriggerNow(context.Background())
	if len(lobbies) != 1 {
		t.Fatalf("expected 1 lobby, got %d", len(lobbies))
	}
	if len(lobbies[0].Teams) != 3 {
		t.Fatalf("expected 3 teams in lobby, got %d", len(lobbies[0].Teams))
	}
	if q.Len() != 0 {
		t.Fatalf("expected 0 leftover, got %d", q.Len())
	}
}

func TestBatcher_MultipleLobbies(t *testing.T) {
	q := NewQueue()
	adapter := mmr.NewMock()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	b := NewBatcher(q, adapter, 3, time.Hour, logger)

	for i := 0; i < 7; i++ {
		q.Enqueue(makeEntry("t"+itoa(i), float64(1000+i*100)))
	}

	lobbies := b.TriggerNow(context.Background())
	if len(lobbies) != 2 {
		t.Fatalf("expected 2 lobbies from 7 teams (lobbySize=3), got %d", len(lobbies))
	}
	// 1 leftover team.
	if q.Len() != 1 {
		t.Fatalf("expected 1 leftover, got %d", q.Len())
	}
}

func TestBatcher_SortsByMMR(t *testing.T) {
	q := NewQueue()
	adapter := mmr.NewMock()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	b := NewBatcher(q, adapter, 3, time.Hour, logger)

	// Enqueue in reverse order.
	q.Enqueue(makeEntry("high", 3000))
	q.Enqueue(makeEntry("low", 1000))
	q.Enqueue(makeEntry("mid", 2000))

	lobbies := b.TriggerNow(context.Background())
	if len(lobbies) != 1 {
		t.Fatalf("expected 1 lobby, got %d", len(lobbies))
	}

	// First team in lobby should be lowest MMR.
	if lobbies[0].Teams[0].MMR > lobbies[0].Teams[2].MMR {
		t.Fatalf("expected sorted order: first=%.0f last=%.0f",
			lobbies[0].Teams[0].MMR, lobbies[0].Teams[2].MMR)
	}
}

func TestBatcher_MMRSpreadAndAvg(t *testing.T) {
	q := NewQueue()
	adapter := mmr.NewMock()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	b := NewBatcher(q, adapter, 3, time.Hour, logger)

	q.Enqueue(makeEntry("a", 1000))
	q.Enqueue(makeEntry("b", 1500))
	q.Enqueue(makeEntry("c", 2000))

	lobbies := b.TriggerNow(context.Background())
	lobby := lobbies[0]

	expectedAvg := (1000.0 + 1500.0 + 2000.0) / 3
	if lobby.AvgMMR != expectedAvg {
		t.Fatalf("expected avg %.2f, got %.2f", expectedAvg, lobby.AvgMMR)
	}
	if lobby.MMRSpread != 1000 {
		t.Fatalf("expected spread 1000, got %.2f", lobby.MMRSpread)
	}
}

func TestBatcher_WithFakeTeams(t *testing.T) {
	q := NewQueue()
	adapter := mmr.NewMock()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	lobbySize := 5
	b := NewBatcher(q, adapter, lobbySize, time.Hour, logger)

	// Use the fake data generator.
	teams := nhost.GenerateFakeTeams("test-scrim", 12)
	ctx := context.Background()
	for _, team := range teams {
		teamMMR, _ := adapter.GetTeamMMR(ctx, team.Players)
		q.Enqueue(models.QueueEntry{
			Team:       team,
			MMR:        teamMMR,
			EnqueuedAt: time.Now().UTC(),
		})
	}

	lobbies := b.TriggerNow(ctx)
	if len(lobbies) != 2 { // 12 teams / 5 = 2 lobbies, 2 leftover
		t.Fatalf("expected 2 lobbies from 12 teams (lobbySize=5), got %d", len(lobbies))
	}
	if q.Len() != 2 {
		t.Fatalf("expected 2 leftover, got %d", q.Len())
	}

	// Verify lobbies are internally sorted by MMR.
	for li, lobby := range lobbies {
		for i := 1; i < len(lobby.Teams); i++ {
			if lobby.Teams[i].MMR < lobby.Teams[i-1].MMR {
				t.Fatalf("lobby %d not sorted at index %d: %.0f < %.0f",
					li, i, lobby.Teams[i].MMR, lobby.Teams[i-1].MMR)
			}
		}
	}
}

func TestBatcher_GetLobbiesCopy(t *testing.T) {
	q := NewQueue()
	adapter := mmr.NewMock()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	b := NewBatcher(q, adapter, 2, time.Hour, logger)

	q.Enqueue(makeEntry("a", 1000))
	q.Enqueue(makeEntry("b", 2000))
	b.TriggerNow(context.Background())

	lobbies := b.GetLobbies()
	if len(lobbies) != 1 {
		t.Fatalf("expected 1 stored lobby, got %d", len(lobbies))
	}

	// Mutating returned slice should not affect internal state.
	lobbies[0].ID = "mutated"
	original := b.GetLobbies()
	if original[0].ID == "mutated" {
		t.Fatal("GetLobbies returned a reference instead of a copy")
	}
}

// ---------- helpers ----------

func makeEntry(name string, mmrVal float64) models.QueueEntry {
	return models.QueueEntry{
		Team: models.Team{
			TeamName: name,
			ScrimID:  "test-scrim",
			Players: [3]models.Player{
				{ID: "p1", DiscordID: "d1", DisplayName: "P1", Elo: mmrVal},
				{ID: "p2", DiscordID: "d2", DisplayName: "P2", Elo: mmrVal},
				{ID: "p3", DiscordID: "d3", DisplayName: "P3", Elo: mmrVal},
			},
		},
		MMR:        mmrVal,
		EnqueuedAt: time.Now().UTC(),
	}
}

// silence the "imported and not used" if tests are skipped
var _ = os.DevNull
