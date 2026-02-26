package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// LLM (eino)
	LLMBaseURL   string // LLM_BASE_URL
	LLMAPIKey    string // LLM_API_KEY
	LLMModelName string // LLM_MODEL_NAME

	// qui API
	QuiBaseURL    string // QUI_BASE_URL
	QuiAPIKey     string // QUI_API_KEY
	QuiInstanceID int    // QUI_INSTANCE_ID (defaults to 1)

	// TMDb
	TMDbAPIKey string // TMDB_API_KEY

	// Anime list API
	AnimeListBaseURL string // ANIMELIST_BASE_URL
	AnimeListUsername string // ANIMELIST_USERNAME
	AnimeListPassword string // ANIMELIST_PASSWORD

	// Webhook
	WebhookURL string // WEBHOOK_URL

	// rsync / NAS
	NASDestPath string // NAS_DEST_PATH (e.g. user@nas:/volume1/anime)
}

var required = []string{
	"LLM_BASE_URL",
	"LLM_API_KEY",
	"LLM_MODEL_NAME",
	"TMDB_API_KEY",
	"QUI_BASE_URL",
	"ANIMELIST_BASE_URL",
	"NAS_DEST_PATH",
}

// Load reads configuration from environment variables and returns a Config.
// Returns an error if any required variable is missing.
func Load() (*Config, error) {
	for _, key := range required {
		if os.Getenv(key) == "" {
			return nil, fmt.Errorf("missing required env var: %s", key)
		}
	}

	quiInstanceID := 1
	if raw := os.Getenv("QUI_INSTANCE_ID"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid QUI_INSTANCE_ID: %w", err)
		}
		quiInstanceID = parsed
	}

	return &Config{
		LLMBaseURL:       os.Getenv("LLM_BASE_URL"),
		LLMAPIKey:        os.Getenv("LLM_API_KEY"),
		LLMModelName:     os.Getenv("LLM_MODEL_NAME"),
		QuiBaseURL:       os.Getenv("QUI_BASE_URL"),
		QuiAPIKey:        os.Getenv("QUI_API_KEY"),
		QuiInstanceID:    quiInstanceID,
		TMDbAPIKey:       os.Getenv("TMDB_API_KEY"),
		AnimeListBaseURL:  os.Getenv("ANIMELIST_BASE_URL"),
		AnimeListUsername: os.Getenv("ANIMELIST_USERNAME"),
		AnimeListPassword: os.Getenv("ANIMELIST_PASSWORD"),
		WebhookURL:       os.Getenv("WEBHOOK_URL"),
		NASDestPath:      os.Getenv("NAS_DEST_PATH"),
	}, nil
}
