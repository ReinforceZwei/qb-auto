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

	// rsync daemon
	RsyncHost         string // RSYNC_HOST          (required) e.g. "192.168.1.100"
	RsyncModule       string // RSYNC_MODULE        (required) rsync module = NAS share root, e.g. "media"
	RsyncUser         string // RSYNC_USER          (required)
	RsyncPasswordFile string // RSYNC_PASSWORD_FILE (required) path to plaintext password file
	RsyncPort         int    // RSYNC_PORT          (optional, default 873)
	RsyncBinaryPath   string // RSYNC_BINARY        (optional, default "rsync")
	NASAnimeBasePath  string // NAS_ANIME_BASE_PATH (required) folder inside the rsync module where anime lives, e.g. "anime"

	// Workers
	TitleWorkerCount int // TITLE_WORKER_COUNT (defaults to 1)
}

var required = []string{
	"LLM_BASE_URL",
	"LLM_API_KEY",
	"LLM_MODEL_NAME",
	"TMDB_API_KEY",
	"QUI_BASE_URL",
	"ANIMELIST_BASE_URL",
	"RSYNC_HOST",
	"RSYNC_MODULE",
	"RSYNC_USER",
	"RSYNC_PASSWORD_FILE",
	"NAS_ANIME_BASE_PATH",
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

	titleWorkerCount := 1
	if raw := os.Getenv("TITLE_WORKER_COUNT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid TITLE_WORKER_COUNT: %w", err)
		}
		if parsed < 1 {
			return nil, fmt.Errorf("TITLE_WORKER_COUNT must be at least 1")
		}
		titleWorkerCount = parsed
	}

	rsyncPort := 873
	if raw := os.Getenv("RSYNC_PORT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid RSYNC_PORT: %w", err)
		}
		rsyncPort = parsed
	}

	rsyncBinaryPath := "rsync"
	if raw := os.Getenv("RSYNC_BINARY"); raw != "" {
		rsyncBinaryPath = raw
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
		WebhookURL:        os.Getenv("WEBHOOK_URL"),
		RsyncHost:         os.Getenv("RSYNC_HOST"),
		RsyncModule:       os.Getenv("RSYNC_MODULE"),
		RsyncUser:         os.Getenv("RSYNC_USER"),
		RsyncPasswordFile: os.Getenv("RSYNC_PASSWORD_FILE"),
		RsyncPort:         rsyncPort,
		RsyncBinaryPath:   rsyncBinaryPath,
		NASAnimeBasePath:  os.Getenv("NAS_ANIME_BASE_PATH"),
		TitleWorkerCount:  titleWorkerCount,
	}, nil
}
