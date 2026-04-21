// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
	"strconv"
)

type Config struct {
	// HTTP listen address.
	Addr string
	// LobbySize is the number of teams per lobby (matches scrim-bot lobbySize).
	LobbySize int
	// WebhookSecret for HMAC signature verification on inbound webhooks.
	WebhookSecret string
	// Nhost connection details (GraphQL endpoint + admin secret).
	NhostGraphQLURL string
	NhostAdminSecret string
	// DevMode disables external calls (nhost, bot notifications).
	// Set DEV_MODE=true to enable.
	DevMode bool
}

func Load() Config {
	c := Config{
		Addr:             envOr("ADDR", ":8080"),
		LobbySize:        envInt("LOBBY_SIZE", 20),
		WebhookSecret:    envOr("WEBHOOK_SECRET", "dev-secret"),
		NhostGraphQLURL:  envOr("NHOST_GRAPHQL_URL", "http://localhost:1337/v1/graphql"),
		NhostAdminSecret: envOr("NHOST_ADMIN_SECRET", "nhost-admin-secret"),
		DevMode:          envBool("DEV_MODE", false),
	}
	return c
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
