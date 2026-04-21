package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/matchmaker"
	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/mmr"
	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/models"
	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/nhost"
)

func newTestServer() *Server {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	queue := matchmaker.NewQueue()
	adapter := mmr.NewMock()
	batcher := matchmaker.NewBatcher(queue, adapter, 3, time.Hour, logger)
	stub := nhost.NewStub(logger, 6)
	return NewServer(queue, batcher, adapter, stub, logger, "dev-secret", true)
}

// ---------- Health ----------

func TestHealth(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Fatalf("expected ok, got %s", body["status"])
	}
}

// ---------- Signup ----------

func TestSignup(t *testing.T) {
	srv := newTestServer()

	payload := SignupPayload{
		TeamName: "TestTeam",
		ScrimID:  "scrim_1",
		SignupPlayer: models.Player{
			ID: "su1", DiscordID: "d_su1", DisplayName: "Captain",
		},
		Players: [3]models.Player{
			{ID: "p1", DiscordID: "d1", DisplayName: "P1", Elo: 1500},
			{ID: "p2", DiscordID: "d2", DisplayName: "P2", Elo: 1600},
			{ID: "p3", DiscordID: "d3", DisplayName: "P3", Elo: 1400},
		},
		SignupTime: time.Now().UTC(),
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	if srv.Queue.Len() != 1 {
		t.Fatalf("expected 1 team in queue, got %d", srv.Queue.Len())
	}
}

func TestSignup_InvalidJSON(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/webhook/signup", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ---------- Cancel ----------

func TestCancel(t *testing.T) {
	srv := newTestServer()

	// Enqueue a team first.
	srv.Queue.Enqueue(models.QueueEntry{
		Team: models.Team{TeamName: "ToCancel", ScrimID: "s1"},
		MMR:  1500,
	})

	body, _ := json.Marshal(map[string]string{"team_name": "ToCancel", "scrim_id": "s1"})
	req := httptest.NewRequest(http.MethodPost, "/webhook/cancel", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if srv.Queue.Len() != 0 {
		t.Fatalf("expected 0 in queue after cancel, got %d", srv.Queue.Len())
	}
}

func TestCancel_NotFound(t *testing.T) {
	srv := newTestServer()

	body, _ := json.Marshal(map[string]string{"team_name": "Ghost", "scrim_id": "s1"})
	req := httptest.NewRequest(http.MethodPost, "/webhook/cancel", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// ---------- Queue Status ----------

func TestQueueStatus(t *testing.T) {
	srv := newTestServer()
	srv.Queue.Enqueue(models.QueueEntry{Team: models.Team{TeamName: "x"}, MMR: 1500})

	req := httptest.NewRequest(http.MethodGet, "/status/queue", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]int
	json.NewDecoder(w.Body).Decode(&body)
	if body["queue_depth"] != 1 {
		t.Fatalf("expected queue_depth=1, got %d", body["queue_depth"])
	}
}

// ---------- Admin: Enqueue Scrim (dev stub) ----------

func TestEnqueueScrim_DevMode(t *testing.T) {
	srv := newTestServer() // uses stub with 6 fake teams

	req := httptest.NewRequest(http.MethodPost, "/admin/enqueue_scrim/dev-scrim-1", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	enqueued := int(body["enqueued"].(float64))
	if enqueued != 6 {
		t.Fatalf("expected 6 enqueued, got %d", enqueued)
	}
	if srv.Queue.Len() != 6 {
		t.Fatalf("expected 6 in queue, got %d", srv.Queue.Len())
	}
}

// ---------- Admin: Batch ----------

func TestBatchNow_NotEnough(t *testing.T) {
	srv := newTestServer()
	srv.Queue.Enqueue(models.QueueEntry{Team: models.Team{TeamName: "lonely"}, MMR: 1500})

	req := httptest.NewRequest(http.MethodPost, "/admin/batch", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if int(body["lobbies_formed"].(float64)) != 0 {
		t.Fatal("expected 0 lobbies when not enough teams")
	}
}

func TestBatchNow_FormsLobby(t *testing.T) {
	srv := newTestServer() // lobbySize=3

	for i := 0; i < 5; i++ {
		srv.Queue.Enqueue(models.QueueEntry{
			Team: models.Team{TeamName: "t" + string(rune('A'+i)), ScrimID: "s1"},
			MMR:  float64(1000 + i*200),
		})
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/batch", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	formed := int(body["lobbies_formed"].(float64))
	if formed != 1 { // 5 teams / 3 = 1 lobby, 2 leftover
		t.Fatalf("expected 1 lobby, got %d", formed)
	}
	depth := int(body["queue_depth"].(float64))
	if depth != 2 {
		t.Fatalf("expected 2 leftover in queue, got %d", depth)
	}
}

// ---------- Lobbies ----------

func TestListLobbies_Empty(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/lobbies", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ---------- Match Result ----------

func TestMatchResult(t *testing.T) {
	srv := newTestServer()

	result := models.MatchResult{
		LobbyID: "lobby_1",
		Teams: []models.TeamPlacement{
			{TeamName: "Alpha", Placement: 1, Score: 20},
			{TeamName: "Beta", Placement: 2, Score: 15},
		},
	}

	body, _ := json.Marshal(result)
	req := httptest.NewRequest(http.MethodPost, "/webhook/match_result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
