package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Config holds all application configuration.
type Config struct {
	// LLM (eino)
	LLMBaseURL   string `json:"llm_base_url"`   // LLM_BASE_URL
	LLMAPIKey    string `json:"llm_api_key"`    // LLM_API_KEY
	LLMModelName string `json:"llm_model_name"` // LLM_MODEL_NAME

	// qui API
	QuiBaseURL    string `json:"qui_base_url"`    // QUI_BASE_URL
	QuiAPIKey     string `json:"qui_api_key"`     // QUI_API_KEY
	QuiInstanceID int    `json:"qui_instance_id"` // QUI_INSTANCE_ID (defaults to 1)

	// TMDb
	TMDbAPIKey string `json:"tmdb_api_key"` // TMDB_API_KEY

	// Anime list API
	AnimeListBaseURL string `json:"animelist_base_url"` // ANIMELIST_BASE_URL
	AnimeListUsername string `json:"animelist_username"` // ANIMELIST_USERNAME
	AnimeListPassword string `json:"animelist_password"` // ANIMELIST_PASSWORD

	// Webhook
	WebhookURL string `json:"webhook_url"` // WEBHOOK_URL

	// rsync daemon
	RsyncHost         string `json:"rsync_host"`          // RSYNC_HOST          (required) e.g. "192.168.1.100"
	RsyncModule       string `json:"rsync_module"`        // RSYNC_MODULE        (required) rsync module = NAS share root, e.g. "media"
	RsyncUser         string `json:"rsync_user"`          // RSYNC_USER          (required)
	RsyncPasswordFile string `json:"rsync_password_file"` // RSYNC_PASSWORD_FILE (required) path to plaintext password file
	RsyncPort         int    `json:"rsync_port"`          // RSYNC_PORT          (optional, default 873)
	RsyncBinaryPath   string `json:"rsync_binary_path"`   // RSYNC_BINARY        (optional, default "rsync")
	NASAnimeBasePath  string `json:"nas_anime_base_path"` // NAS_ANIME_BASE_PATH (required) folder inside the rsync module where anime lives, e.g. "anime"

	// Workers
	TitleWorkerCount int `json:"title_worker_count"` // TITLE_WORKER_COUNT (defaults to 1)

	// PocketBase server
	HttpAddr string `json:"http_addr"` // HTTP_ADDR (optional, default "127.0.0.1:8090")
}

var required = []string{
	"LLMBaseURL",
	"LLMAPIKey",
	"LLMModelName",
	"TMDbAPIKey",
	"QuiBaseURL",
	"AnimeListBaseURL",
	"RsyncHost",
	"RsyncModule",
	"RsyncUser",
	"RsyncPasswordFile",
	"NASAnimeBasePath",
}

// requiredFields maps field name to its string value for validation.
func requiredFields(c *Config) map[string]string {
	return map[string]string{
		"LLMBaseURL":        c.LLMBaseURL,
		"LLMAPIKey":         c.LLMAPIKey,
		"LLMModelName":      c.LLMModelName,
		"TMDbAPIKey":        c.TMDbAPIKey,
		"QuiBaseURL":        c.QuiBaseURL,
		"AnimeListBaseURL":  c.AnimeListBaseURL,
		"RsyncHost":         c.RsyncHost,
		"RsyncModule":       c.RsyncModule,
		"RsyncUser":         c.RsyncUser,
		"RsyncPasswordFile": c.RsyncPasswordFile,
		"NASAnimeBasePath":  c.NASAnimeBasePath,
	}
}

// ConfigPath returns the path to the user config file: ~/.config/qb-auto/config.json
func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine user config directory: %w", err)
	}
	return filepath.Join(dir, "qb-auto", "config.json"), nil
}

// InitConfig creates the config directory and writes a template config.json at path.
func InitConfig(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

	template := &Config{
		QuiInstanceID:    1,
		RsyncPort:        873,
		RsyncBinaryPath:  "rsync",
		TitleWorkerCount: 1,
		HttpAddr:         "127.0.0.1:8090",
	}

	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal default config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("cannot write config file: %w", err)
	}
	return nil
}

// LoadFromFile reads config from the JSON file at path, then applies any environment
// variable overrides on top (env vars take priority over JSON values).
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}

	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("cannot parse config file: %w", err)
	}

	// Apply environment variable overrides for all string fields.
	applyEnvString(&cfg.LLMBaseURL, "LLM_BASE_URL")
	applyEnvString(&cfg.LLMAPIKey, "LLM_API_KEY")
	applyEnvString(&cfg.LLMModelName, "LLM_MODEL_NAME")
	applyEnvString(&cfg.QuiBaseURL, "QUI_BASE_URL")
	applyEnvString(&cfg.QuiAPIKey, "QUI_API_KEY")
	applyEnvString(&cfg.TMDbAPIKey, "TMDB_API_KEY")
	applyEnvString(&cfg.AnimeListBaseURL, "ANIMELIST_BASE_URL")
	applyEnvString(&cfg.AnimeListUsername, "ANIMELIST_USERNAME")
	applyEnvString(&cfg.AnimeListPassword, "ANIMELIST_PASSWORD")
	applyEnvString(&cfg.WebhookURL, "WEBHOOK_URL")
	applyEnvString(&cfg.RsyncHost, "RSYNC_HOST")
	applyEnvString(&cfg.RsyncModule, "RSYNC_MODULE")
	applyEnvString(&cfg.RsyncUser, "RSYNC_USER")
	applyEnvString(&cfg.RsyncPasswordFile, "RSYNC_PASSWORD_FILE")
	applyEnvString(&cfg.RsyncBinaryPath, "RSYNC_BINARY")
	applyEnvString(&cfg.NASAnimeBasePath, "NAS_ANIME_BASE_PATH")
	applyEnvString(&cfg.HttpAddr, "HTTP_ADDR")

	// Apply environment variable overrides for int fields.
	if raw := os.Getenv("QUI_INSTANCE_ID"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid QUI_INSTANCE_ID: %w", err)
		}
		cfg.QuiInstanceID = parsed
	}
	if raw := os.Getenv("TITLE_WORKER_COUNT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid TITLE_WORKER_COUNT: %w", err)
		}
		cfg.TitleWorkerCount = parsed
	}
	if raw := os.Getenv("RSYNC_PORT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid RSYNC_PORT: %w", err)
		}
		cfg.RsyncPort = parsed
	}

	// Apply defaults for int fields that were not set by JSON or env var.
	if cfg.QuiInstanceID == 0 {
		cfg.QuiInstanceID = 1
	}
	if cfg.TitleWorkerCount == 0 {
		cfg.TitleWorkerCount = 1
	} else if cfg.TitleWorkerCount < 1 {
		return nil, fmt.Errorf("title_worker_count must be at least 1")
	}
	if cfg.RsyncPort == 0 {
		cfg.RsyncPort = 873
	}
	if cfg.RsyncBinaryPath == "" {
		cfg.RsyncBinaryPath = "rsync"
	}

	// Validate required fields.
	fields := requiredFields(cfg)
	for _, name := range required {
		if fields[name] == "" {
			return nil, fmt.Errorf("missing required config field: %s", name)
		}
	}

	return cfg, nil
}

func applyEnvString(field *string, key string) {
	if v := os.Getenv(key); v != "" {
		*field = v
	}
}
