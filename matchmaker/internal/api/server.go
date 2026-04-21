// Package api provides HTTP handlers for the matchmaker.
package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/matchmaker"
	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/mmr"
	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/models"
	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/nhost"
)

// Server wraps the HTTP router and dependencies.
type Server struct {
	Router    chi.Router
	Queue     *matchmaker.Queue
	Batcher   *matchmaker.Batcher
	Adapter   mmr.Adapter
	Nhost     nhost.DataSource
	Logger    *slog.Logger
	HMACKey   []byte
	DevMode   bool
}

func NewServer(queue *matchmaker.Queue, batcher *matchmaker.Batcher, adapter mmr.Adapter, nhostDS nhost.DataSource, logger *slog.Logger, hmacKey string, devMode bool) *Server {
	s := &Server{
		Queue:   queue,
		Batcher: batcher,
		Adapter: adapter,
		Nhost:   nhostDS,
		Logger:  logger,
		HMACKey: []byte(hmacKey),
		DevMode: devMode,
	}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", s.handleHealth)

	// Inbound webhooks from bot (HMAC-protected).
	r.Route("/webhook", func(r chi.Router) {
		r.Use(s.hmacMiddleware)
		r.Post("/signup", s.handleSignup)
		r.Post("/cancel", s.handleCancel)
		r.Post("/match_result", s.handleMatchResult)
	})

	// Queue / lobby read endpoints.
	r.Get("/status/queue", s.handleQueueStatus)
	r.Get("/lobbies", s.handleListLobbies)

	// Admin: pull signups directly from nhost and enqueue.
	r.Post("/admin/enqueue_scrim/{scrimID}", s.handleEnqueueScrim)

	// Admin: trigger one batch cycle.
	r.Post("/admin/batch", s.handleBatchNow)

	s.Router = r
	return s
}

// ---------- handlers ----------

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SignupPayload matches what the bot should POST.
type SignupPayload struct {
	TeamName    string           `json:"team_name"`
	ScrimID     string           `json:"scrim_id"`
	SignupPlayer models.Player   `json:"signup_player"`
	Players     [3]models.Player `json:"players"`
	SignupTime  time.Time        `json:"signup_time"`
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	var p SignupPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	team := models.Team{
		TeamName:    p.TeamName,
		ScrimID:     p.ScrimID,
		SignupPlayer: p.SignupPlayer,
		Players:     p.Players,
		SignupTime:  p.SignupTime,
	}

	teamMMR, err := s.Adapter.GetTeamMMR(r.Context(), team.Players)
	if err != nil {
		s.Logger.Warn("mmr lookup failed, using default", "team", team.TeamName, "err", err)
		teamMMR = 1500
	}

	entry := models.QueueEntry{
		Team:       team,
		MMR:        teamMMR,
		EnqueuedAt: time.Now().UTC(),
	}
	s.Queue.Enqueue(entry)
	s.Logger.Info("team enqueued", "team", team.TeamName, "mmr", teamMMR, "queue_depth", s.Queue.Len())

	writeJSON(w, http.StatusAccepted, map[string]any{
		"team":        team.TeamName,
		"mmr":         teamMMR,
		"queue_depth": s.Queue.Len(),
	})
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	var p struct {
		TeamName string `json:"team_name"`
		ScrimID  string `json:"scrim_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	// Remove from queue (linear scan is fine for prototype).
	removed := false
	entries := s.Queue.Drain()
	for _, e := range entries {
		if e.Team.TeamName == p.TeamName && e.Team.ScrimID == p.ScrimID {
			removed = true
			continue
		}
		s.Queue.Enqueue(e)
	}
	if removed {
		s.Logger.Info("team removed from queue", "team", p.TeamName)
		writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
	} else {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "team not in queue"})
	}
}

func (s *Server) handleMatchResult(w http.ResponseWriter, r *http.Request) {
	var result models.MatchResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if err := s.Adapter.UpdateResults(r.Context(), result); err != nil {
		s.Logger.Error("mmr update failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mmr update failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}

func (s *Server) handleQueueStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]int{"queue_depth": s.Queue.Len()})
}

func (s *Server) handleListLobbies(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.Batcher.GetLobbies())
}

// handleEnqueueScrim pulls signups from nhost for a scrim and enqueues them all.
func (s *Server) handleEnqueueScrim(w http.ResponseWriter, r *http.Request) {
	scrimID := chi.URLParam(r, "scrimID")
	teams, err := s.Nhost.GetSignupsForScrim(r.Context(), scrimID)
	if err != nil {
		s.Logger.Error("nhost fetch failed", "scrim_id", scrimID, "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "nhost fetch failed"})
		return
	}

	ctx := r.Context()
	for _, team := range teams {
		teamMMR, err := s.Adapter.GetTeamMMR(ctx, team.Players)
		if err != nil {
			teamMMR = 1500
		}
		s.Queue.Enqueue(models.QueueEntry{
			Team:       team,
			MMR:        teamMMR,
			EnqueuedAt: time.Now().UTC(),
		})
	}
	s.Logger.Info("enqueued teams from nhost", "scrim_id", scrimID, "count", len(teams), "queue_depth", s.Queue.Len())

	writeJSON(w, http.StatusOK, map[string]any{
		"scrim_id":    scrimID,
		"enqueued":    len(teams),
		"queue_depth": s.Queue.Len(),
	})
}

// handleBatchNow forces one batch cycle immediately.
func (s *Server) handleBatchNow(w http.ResponseWriter, r *http.Request) {
	lobbies := s.Batcher.TriggerNow(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"lobbies_formed": len(lobbies),
		"lobbies":        lobbies,
		"queue_depth":    s.Queue.Len(),
	})
}

// ---------- HMAC middleware ----------

func (s *Server) hmacMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only bypass HMAC verification when DEV_MODE=true is set explicitly.
		// A missing or default webhook secret outside dev mode is a
		// configuration error and must be treated as unauthorized, NOT as
		// an implicit bypass (which the previous behaviour allowed).
		if s.DevMode {
			next.ServeHTTP(w, r)
			return
		}
		if len(s.HMACKey) == 0 || string(s.HMACKey) == "dev-secret" {
			s.Logger.Error("webhook rejected: WEBHOOK_SECRET is missing or uses the dev default; refusing request")
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "webhook secret not configured"})
			return
		}

		sig := r.Header.Get("X-Matchmaker-Signature")
		if sig == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing signature"})
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot read body"})
			return
		}
		// Restore body for downstream handlers.
		r.Body = io.NopCloser(bytes.NewReader(body))

		mac := hmac.New(sha256.New, s.HMACKey)
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(sig), []byte(expected)) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid signature"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
