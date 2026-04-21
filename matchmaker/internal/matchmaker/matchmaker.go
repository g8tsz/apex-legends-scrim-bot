// Package matchmaker implements the team queue and lobby balancer.
package matchmaker

import (
	"context"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/mmr"
	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/models"
)

// Queue holds teams waiting to be assigned to a lobby.
type Queue struct {
	mu      sync.Mutex
	entries []models.QueueEntry
}

func NewQueue() *Queue {
	return &Queue{}
}

// Enqueue adds a team to the queue.
func (q *Queue) Enqueue(entry models.QueueEntry) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.entries = append(q.entries, entry)
}

// Len returns current queue depth.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.entries)
}

// Drain removes and returns all entries (used by the batcher).
func (q *Queue) Drain() []models.QueueEntry {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := q.entries
	q.entries = nil
	return out
}

// Batcher periodically forms balanced lobbies from the queue.
type Batcher struct {
	queue     *Queue
	adapter   mmr.Adapter
	lobbySize int // teams per lobby
	interval  time.Duration
	logger    *slog.Logger

	mu      sync.Mutex
	lobbies []models.Lobby
}

func NewBatcher(queue *Queue, adapter mmr.Adapter, lobbySize int, interval time.Duration, logger *slog.Logger) *Batcher {
	return &Batcher{
		queue:     queue,
		adapter:   adapter,
		lobbySize: lobbySize,
		interval:  interval,
		logger:    logger,
	}
}

// Run starts the batcher loop. Cancel the context to stop.
func (b *Batcher) Run(ctx context.Context) {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.tick(ctx)
		}
	}
}

// TriggerNow runs one batch cycle immediately (useful for the API).
func (b *Batcher) TriggerNow(ctx context.Context) []models.Lobby {
	return b.tick(ctx)
}

func (b *Batcher) tick(ctx context.Context) []models.Lobby {
	entries := b.queue.Drain()
	if len(entries) < b.lobbySize {
		// Not enough teams; re-enqueue them.
		for _, e := range entries {
			b.queue.Enqueue(e)
		}
		return nil
	}

	// Ensure every entry has an MMR value.
	for i := range entries {
		if entries[i].MMR == 0 {
			val, err := b.adapter.GetTeamMMR(ctx, entries[i].Team.Players)
			if err != nil {
				b.logger.Warn("mmr lookup failed, using default", "team", entries[i].Team.TeamName, "err", err)
				entries[i].MMR = 1500
			} else {
				entries[i].MMR = val
			}
		}
	}

	lobbies := b.balance(entries)

	b.mu.Lock()
	b.lobbies = append(b.lobbies, lobbies...)
	b.mu.Unlock()

	b.logger.Info("formed lobbies", "count", len(lobbies), "leftover", b.queue.Len())
	return lobbies
}

// balance groups entries into lobbies of lobbySize, minimising MMR spread.
// Algorithm: sort by MMR then take consecutive slices (greedy bucket fill).
// Leftover teams are re-enqueued.
func (b *Batcher) balance(entries []models.QueueEntry) []models.Lobby {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].MMR < entries[j].MMR
	})

	var lobbies []models.Lobby
	for len(entries) >= b.lobbySize {
		bucket := entries[:b.lobbySize]
		entries = entries[b.lobbySize:]

		lobby := models.Lobby{
			ID:        generateID(),
			ScrimID:   bucket[0].Team.ScrimID,
			Teams:     make([]models.QueueEntry, len(bucket)),
			CreatedAt: time.Now().UTC(),
		}
		copy(lobby.Teams, bucket)

		var sum float64
		minMMR := math.MaxFloat64
		maxMMR := -math.MaxFloat64
		for _, e := range bucket {
			sum += e.MMR
			if e.MMR < minMMR {
				minMMR = e.MMR
			}
			if e.MMR > maxMMR {
				maxMMR = e.MMR
			}
		}
		lobby.AvgMMR = sum / float64(len(bucket))
		lobby.MMRSpread = maxMMR - minMMR

		lobbies = append(lobbies, lobby)
	}

	// Re-enqueue leftover teams.
	for _, e := range entries {
		b.queue.Enqueue(e)
	}

	return lobbies
}

// GetLobbies returns all lobbies formed so far.
func (b *Batcher) GetLobbies() []models.Lobby {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]models.Lobby, len(b.lobbies))
	copy(out, b.lobbies)
	return out
}

// simple incrementing ID for prototype; replace with UUID in production.
var idCounter int
var idMu sync.Mutex

func generateID() string {
	idMu.Lock()
	defer idMu.Unlock()
	idCounter++
	return "lobby_" + time.Now().Format("20060102") + "_" + itoa(idCounter)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
