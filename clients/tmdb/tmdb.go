package tmdb

import (
	"fmt"

	"github.com/ReinforceZwei/qb-auto/config"
	gotmdb "github.com/cyruzin/golang-tmdb"
)

// Candidate represents a TMDb TV show search result reduced to the fields
// needed for LLM matching and title lookup.
type Candidate struct {
	ID           int
	Name         string
	OriginalName string
	Overview     string
}

// Client is a thin wrapper around the golang-tmdb client.
type Client struct {
	inner *gotmdb.Client
}

// New initialises a Client using the TMDb API key from cfg.
func New(cfg *config.Config) (*Client, error) {
	c, err := gotmdb.Init(cfg.TMDbAPIKey)
	if err != nil {
		return nil, fmt.Errorf("tmdb: init client: %w", err)
	}
	return &Client{inner: c}, nil
}

// SearchAnime searches TMDb for TV shows matching title, requesting results in
// Traditional Chinese (zh-TW). Returns a slice of Candidates (may be empty).
func (c *Client) SearchAnime(title string) ([]Candidate, error) {
	results, err := c.inner.GetSearchTVShow(title, map[string]string{
		"language": "zh-TW",
	})
	if err != nil {
		return nil, fmt.Errorf("tmdb: search anime %q: %w", title, err)
	}
	if results.SearchTVShowsResults == nil {
		return nil, nil
	}

	candidates := make([]Candidate, 0, len(results.Results))
	for _, r := range results.Results {
		candidates = append(candidates, Candidate{
			ID:           int(r.ID),
			Name:         r.Name,
			OriginalName: r.OriginalName,
			Overview:     r.Overview,
		})
	}
	return candidates, nil
}

// GetTraditionalChineseTitle fetches the zh-TW translated name for the given
// TMDb TV show ID. Returns an error if no zh-TW translation exists.
func (c *Client) GetTraditionalChineseTitle(tvID int) (string, error) {
	translations, err := c.inner.GetTVTranslations(tvID, nil)
	if err != nil {
		return "", fmt.Errorf("tmdb: get translations for id=%d: %w", tvID, err)
	}

	for _, t := range translations.Translations {
		if t.Iso639_1 == "zh" && t.Iso3166_1 == "TW" {
			name := t.Data.Name
			if name == "" {
				name = t.Data.Title
			}
			if name == "" {
				return "", fmt.Errorf("tmdb: zh-TW translation found but name is empty for id=%d", tvID)
			}
			return name, nil
		}
	}
	return "", fmt.Errorf("tmdb: no zh-TW translation found for tv id=%d", tvID)
}
