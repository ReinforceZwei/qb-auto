package animelistnext

import (
	"fmt"
	"sync"
	"time"

	"github.com/ReinforceZwei/qb-auto/config"
	"resty.dev/v3"
)

// tokenExpiry is the assumed lifetime of a PocketBase auth token. PocketBase
// default is 30 days, but we refresh proactively after 23 hours to be safe.
const tokenExpiry = 23 * time.Hour

// AnimeRecord represents a record from the animes collection.
type AnimeRecord struct {
	ID               string `json:"id"`
	TMDbID           int    `json:"tmdbId"`
	TMDbSeasonNum    int    `json:"tmdbSeasonNumber"`
	TMDbMediaType    string `json:"tmdbMediaType"`
	CustomName       string `json:"customName"`
	CachedTitle      string `json:"cachedTitle"`
	CachedSeasonName string `json:"cachedSeasonName"`
	DownloadStatus   string `json:"downloadStatus"`
}

type authResponse struct {
	Token string `json:"token"`
}

type recordListResponse struct {
	Items []AnimeRecord `json:"items"`
}

// Client is a PocketBase REST API client for Anime List Next.
// It authenticates via the users collection and queries the animes collection.
type Client struct {
	baseURL  string
	username string
	password string
	http     *resty.Client

	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

// New creates a Client from cfg and performs an initial login.
// baseURL should point to the root of the PocketBase instance,
// e.g. "https://animelist.example.com".
func New(cfg *config.Config) (*Client, error) {
	rc := resty.New().SetBaseURL(cfg.AnimeListBaseURL)
	c := &Client{
		baseURL:  cfg.AnimeListBaseURL,
		username: cfg.AnimeListUsername,
		password: cfg.AnimeListPassword,
		http:     rc,
	}
	if err := c.login(); err != nil {
		return nil, err
	}
	return c, nil
}

// login authenticates against the users collection and stores the token.
func (c *Client) login() error {
	var result authResponse
	resp, err := c.http.R().
		SetBody(map[string]string{
			"identity": c.username,
			"password": c.password,
		}).
		SetResult(&result).
		Post("/api/collections/users/auth-with-password")
	if err != nil {
		return fmt.Errorf("animelistnext: login: %w", err)
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("animelistnext: login: status %d", resp.StatusCode())
	}
	if result.Token == "" {
		return fmt.Errorf("animelistnext: login: empty token in response")
	}

	c.mu.Lock()
	c.token = result.Token
	c.tokenExp = time.Now().Add(tokenExpiry)
	c.mu.Unlock()
	return nil
}

// currentToken returns the cached token, re-authenticating if it is expired.
func (c *Client) currentToken() (string, error) {
	c.mu.Lock()
	expired := time.Now().After(c.tokenExp)
	c.mu.Unlock()

	if expired {
		if err := c.login(); err != nil {
			return "", err
		}
	}

	c.mu.Lock()
	tok := c.token
	c.mu.Unlock()
	return tok, nil
}

// doWithAuth executes fn with the current token. If the server returns 401 it
// re-authenticates once and retries.
func (c *Client) doWithAuth(fn func(token string) (*resty.Response, error)) (*resty.Response, error) {
	tok, err := c.currentToken()
	if err != nil {
		return nil, err
	}

	resp, err := fn(tok)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == 401 {
		if loginErr := c.login(); loginErr != nil {
			return nil, fmt.Errorf("animelistnext: re-login failed: %w", loginErr)
		}
		c.mu.Lock()
		tok = c.token
		c.mu.Unlock()
		return fn(tok)
	}

	return resp, nil
}

// FindByTMDb queries the animes collection for a record matching the given
// tmdbId and tmdbSeasonNumber. Returns (nil, nil) when no record is found.
//
// When seasonNumber is 1 (the LLM default when it can't identify a season)
// and the S1 record is already downloaded, it falls back to the first
// undownloaded record sharing the same tmdbId (sorted by season ascending).
// This handles e.g. "SHOW BY ROCK S2" where the LLM falls back to season 1
// but S1 is already downloaded and S2 is the real target.
func (c *Client) FindByTMDb(tmdbID int, seasonNumber int) (*AnimeRecord, error) {
	// 1. Try exact (tmdbId + seasonNumber) match.
	record, err := c.queryOne(fmt.Sprintf("(tmdbId = %d && tmdbSeasonNumber = %d)", tmdbID, seasonNumber))
	if err != nil {
		return nil, err
	}

	// No record at all.
	if record == nil {
		return nil, nil
	}

	// Happy path: not downloaded yet.
	if record.DownloadStatus != "downloaded" {
		return record, nil
	}

	// Record is already downloaded. Only override when seasonNumber is 1 —
	// the LLM default that can't be trusted. For any other season the LLM
	// explicitly identified it, so we return the match as-is.
	if seasonNumber != 1 {
		return record, nil
	}

	// 2. S1 is downloaded — look for the first undownloaded season (S2+).
	all, err := c.queryAll(fmt.Sprintf("(tmdbId = %d)", tmdbID), "tmdbSeasonNumber")
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].DownloadStatus != "downloaded" {
			return &all[i], nil
		}
	}

	// 3. All seasons are downloaded — return S1 as-is (idempotent re-mark).
	return record, nil
}

// queryOne fetches at most one record matching the PocketBase filter.
func (c *Client) queryOne(filter string) (*AnimeRecord, error) {
	var result recordListResponse
	resp, err := c.doWithAuth(func(token string) (*resty.Response, error) {
		return c.http.R().
			SetHeader("Authorization", token).
			SetQueryParam("filter", filter).
			SetQueryParam("perPage", "1").
			SetResult(&result).
			Get("/api/collections/animes/records")
	})
	if err != nil {
		return nil, fmt.Errorf("animelistnext: query one: %w", err)
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("animelistnext: query one: status %d", resp.StatusCode())
	}
	if len(result.Items) == 0 {
		return nil, nil
	}
	return &result.Items[0], nil
}

// queryAll fetches all records matching the PocketBase filter (up to 50),
// sorted by the given sort field (e.g. "tmdbSeasonNumber").
func (c *Client) queryAll(filter string, sort string) ([]AnimeRecord, error) {
	var result recordListResponse
	resp, err := c.doWithAuth(func(token string) (*resty.Response, error) {
		return c.http.R().
			SetHeader("Authorization", token).
			SetQueryParam("filter", filter).
			SetQueryParam("sort", sort).
			SetQueryParam("perPage", "50").
			SetResult(&result).
			Get("/api/collections/animes/records")
	})
	if err != nil {
		return nil, fmt.Errorf("animelistnext: query all: %w", err)
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("animelistnext: query all: status %d", resp.StatusCode())
	}
	return result.Items, nil
}

// MarkDownloaded sets downloadStatus = "downloaded" on the animes record with
// the given PocketBase record ID.
func (c *Client) MarkDownloaded(id string) error {
	resp, err := c.doWithAuth(func(token string) (*resty.Response, error) {
		return c.http.R().
			SetHeader("Authorization", token).
			SetBody(map[string]string{"downloadStatus": "downloaded"}).
			Patch("/api/collections/animes/records/" + id)
	})
	if err != nil {
		return fmt.Errorf("animelistnext: mark downloaded: %w", err)
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("animelistnext: mark downloaded: status %d", resp.StatusCode())
	}
	return nil
}
