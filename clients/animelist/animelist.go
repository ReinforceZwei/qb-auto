package animelist

import (
	"fmt"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ReinforceZwei/qb-auto/config"
	"resty.dev/v3"
)

const cacheTTL = 5 * time.Minute

// AnimeRecord represents a single entry returned by the anime list API.
// Field names and types match the server's to_client_dict() output exactly.
type AnimeRecord struct {
	ID          int      `json:"animeID"`
	Name        string   `json:"animeName"`
	AddedTime   int64    `json:"addedTime"`
	WatchedTime int64    `json:"watchedTime"`
	Downloaded  int      `json:"downloaded"` // 1 = downloaded, 0 = not
	Watched     int      `json:"watched"`    // 1 = watched, 0 = not
	Rating      int      `json:"rating"`
	Comment     string   `json:"comment"`
	URL         string   `json:"url"`
	Remark      string   `json:"remark"`
	Tags        []string `json:"tags"`
}

// IsDownloaded reports whether the record has been marked downloaded.
func (r AnimeRecord) IsDownloaded() bool { return r.Downloaded == 1 }

// IsWatched reports whether the record has been marked watched.
func (r AnimeRecord) IsWatched() bool { return r.Watched == 1 }

// Client is an anime list API client that uses cookie-based session auth.
type Client struct {
	baseURL  string
	username string
	password string
	http     *resty.Client

	mu          sync.Mutex
	cachedAll   []AnimeRecord
	cacheExpiry time.Time
}

// New creates a Client from cfg and performs an initial login to obtain session cookies.
func New(cfg *config.Config) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("animelist: create cookie jar: %w", err)
	}

	rc := resty.New().
		SetBaseURL(cfg.AnimeListBaseURL).
		SetCookieJar(jar)

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

// login posts credentials to /login and verifies the session cookie was granted.
func (c *Client) login() error {
	_, err := c.http.R().
		SetFormData(map[string]string{
			"name":     c.username,
			"password": c.password,
		}).
		Post("/login")
	if err != nil {
		return fmt.Errorf("animelist: login: %w", err)
	}

	// A failed login returns 200 + HTML with no cookie set.
	// A successful login issues a 302 → / with Set-Cookie, then resty follows to /.
	base, _ := url.Parse(c.baseURL)
	for _, cookie := range c.http.CookieJar().Cookies(base) {
		if cookie.Name == "animelist_access_token" && cookie.Value != "" {
			return nil
		}
	}
	return fmt.Errorf("animelist: login failed: invalid credentials")
}

// isHTML returns true when the response body is HTML, signalling that the request
// was silently redirected to the login page because the session expired.
func isHTML(resp *resty.Response) bool {
	return strings.Contains(resp.Header().Get("Content-Type"), "text/html")
}

// withAuth executes fn and, if the response looks like a login-page redirect,
// re-authenticates once and retries. The cache is also invalidated on re-login.
func (c *Client) withAuth(fn func() (*resty.Response, error)) (*resty.Response, error) {
	resp, err := fn()
	if err != nil {
		return nil, err
	}
	if !isHTML(resp) {
		return resp, nil
	}

	c.mu.Lock()
	loginErr := c.login()
	c.cachedAll = nil // stale after re-login
	c.mu.Unlock()

	if loginErr != nil {
		return nil, fmt.Errorf("animelist: re-login failed: %w", loginErr)
	}
	return fn()
}

// getAll fetches all records from /get, using a short-lived in-memory cache to
// avoid redundant round trips when Search is called multiple times in a workflow.
func (c *Client) getAll() ([]AnimeRecord, error) {
	c.mu.Lock()
	if c.cachedAll != nil && time.Now().Before(c.cacheExpiry) {
		cached := c.cachedAll
		c.mu.Unlock()
		return cached, nil
	}
	c.mu.Unlock()

	var records []AnimeRecord
	resp, err := c.withAuth(func() (*resty.Response, error) {
		return c.http.R().
			SetResult(&records).
			Get("/get")
	})
	if err != nil {
		return nil, fmt.Errorf("animelist: get all: %w", err)
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("animelist: get all: unexpected status %d", resp.StatusCode())
	}

	c.mu.Lock()
	c.cachedAll = records
	c.cacheExpiry = time.Now().Add(cacheTTL)
	c.mu.Unlock()

	return records, nil
}

// InvalidateCache discards the cached record list, forcing the next Search to
// fetch fresh data from the server.
func (c *Client) InvalidateCache() {
	c.mu.Lock()
	c.cachedAll = nil
	c.mu.Unlock()
}

// Search returns all records whose name contains query (case-insensitive partial match).
func (c *Client) Search(query string) ([]AnimeRecord, error) {
	all, err := c.getAll()
	if err != nil {
		return nil, fmt.Errorf("animelist: search: %w", err)
	}

	lower := strings.ToLower(query)
	var matches []AnimeRecord
	for _, r := range all {
		if strings.Contains(strings.ToLower(r.Name), lower) {
			matches = append(matches, r)
		}
	}
	return matches, nil
}

// GetUnwatchedUndownloaded returns all records where watched == 0 AND downloaded == 0,
// sorted by addedTime descending (most recently added first).
func (c *Client) GetUnwatchedUndownloaded() ([]AnimeRecord, error) {
	all, err := c.getAll()
	if err != nil {
		return nil, fmt.Errorf("animelist: get unwatched undownloaded: %w", err)
	}

	var result []AnimeRecord
	for _, r := range all {
		if r.Watched == 0 && r.Downloaded == 0 {
			result = append(result, r)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].AddedTime > result[j].AddedTime
	})

	return result, nil
}

// MarkDownloaded sets the downloaded flag on the record with the given ID.
// If tmdbID is non-zero, the TMDb TV show page URL is also written to the url
// field in the same request, avoiding a separate round trip.
func (c *Client) MarkDownloaded(id int, tmdbID int) error {
	fields := map[string]string{"downloaded": "true"}
	if tmdbID != 0 {
		fields["url"] = fmt.Sprintf("https://www.themoviedb.org/tv/%d", tmdbID)
	}

	resp, err := c.withAuth(func() (*resty.Response, error) {
		return c.http.R().
			SetFormData(fields).
			Post("/update/" + strconv.Itoa(id))
	})
	if err != nil {
		return fmt.Errorf("animelist: mark downloaded: %w", err)
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("animelist: mark downloaded: unexpected status %d", resp.StatusCode())
	}

	c.InvalidateCache()
	return nil
}
