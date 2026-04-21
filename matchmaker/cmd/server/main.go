// Entrypoint for the matchmaker service.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/api"
	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/config"
	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/matchmaker"
	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/mmr"
	"github.com/g8tsz/apex-legends-scrim-bot/matchmaker/internal/nhost"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()

	// Pluggable MMR adapter — swap with a real one later.
	adapter := mmr.NewMock()

	// Core matchmaker queue + batcher.
	queue := matchmaker.NewQueue()
	batcher := matchmaker.NewBatcher(queue, adapter, cfg.LobbySize, 15*time.Second, logger)

	// Nhost data source: real client or dev stub.
	var nhostDS nhost.DataSource
	if cfg.DevMode {
		logger.Info("DEV MODE enabled — using stub nhost data source, no external calls")
		nhostDS = nhost.NewStub(logger, cfg.LobbySize*2) // generate enough teams for 2 lobbies
	} else {
		nhostDS = nhost.NewClient(cfg.NhostGraphQLURL, cfg.NhostAdminSecret)
	}

	// HTTP server.
	srv := api.NewServer(queue, batcher, adapter, nhostDS, logger, cfg.WebhookSecret, cfg.DevMode)

	httpServer := &http.Server{
		Addr:    cfg.Addr,
		Handler: srv.Router,
	}

	// Start batcher in background.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go batcher.Run(ctx)

	logger.Info("matchmaker starting", "addr", cfg.Addr, "lobby_size", cfg.LobbySize, "dev_mode", cfg.DevMode)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	httpServer.Shutdown(shutdownCtx)
}
