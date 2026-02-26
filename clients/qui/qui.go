package qui

import (
	"encoding/json"
	"fmt"

	"github.com/ReinforceZwei/qb-auto/config"
	"resty.dev/v3"
)

// torrentFilter is the filter object for the torrents list API.
type torrentFilter struct {
	Expr             string   `json:"expr"`
	Status           []string `json:"status"`
	ExcludeStatus    []string `json:"excludeStatus"`
	Categories       []string `json:"categories"`
	ExcludeCategories []string `json:"excludeCategories"`
	Tags             []string `json:"tags"`
	ExcludeTags      []string `json:"excludeTags"`
	Trackers         []string `json:"trackers"`
	ExcludeTrackers  []string `json:"excludeTrackers"`
}

// TorrentInfo holds the fields relevant to the workflow from the qui torrents response.
type TorrentInfo struct {
	Hash        string `json:"hash"`
	Name        string `json:"name"`
	ContentPath string `json:"content_path"`
	SavePath    string `json:"save_path"`
	Category    string `json:"category"`
	Tags        string `json:"tags"`
	State       string `json:"state"`
}

// TorrentListResponse is the response from GET /api/instances/:id/torrents.
type TorrentListResponse struct {
	Torrents []TorrentInfo `json:"torrents"`
	Total    int           `json:"total"`
}

// bulkActionPayload is the request body for POST /api/instances/:id/torrents/bulk-action.
type bulkActionPayload struct {
	Hashes    []string `json:"hashes"`
	Action    string   `json:"action"`
	Tags      string   `json:"tags"`
	SelectAll bool     `json:"selectAll"`
}

// Client is a thin HTTP client for the qui API.
type Client struct {
	rc         *resty.Client
	instanceID int
}

// New creates a new Client configured from cfg.
func New(cfg *config.Config) *Client {
	rc := resty.New().
		SetBaseURL(cfg.QuiBaseURL).
		SetHeader("X-API-Key", cfg.QuiAPIKey)
	return &Client{rc: rc, instanceID: cfg.QuiInstanceID}
}

// GetTorrent retrieves torrent details for the given hash.
// Returns nil, nil if no torrent is found.
func (c *Client) GetTorrent(hash string) (*TorrentInfo, error) {
	filter := torrentFilter{
		Expr:              fmt.Sprintf(`Hash == "%s"`, hash),
		Status:            []string{},
		ExcludeStatus:     []string{},
		Categories:        []string{},
		ExcludeCategories: []string{},
		Tags:              []string{},
		ExcludeTags:       []string{},
		Trackers:          []string{},
		ExcludeTrackers:   []string{},
	}

	filterJSON, err := json.Marshal(filter)
	if err != nil {
		return nil, fmt.Errorf("qui: marshal filter: %w", err)
	}

	var result TorrentListResponse
	resp, err := c.rc.R().
		SetQueryParam("limit", "1").
		SetQueryParam("filters", string(filterJSON)).
		SetResult(&result).
		Get(fmt.Sprintf("/api/instances/%d/torrents", c.instanceID))
	if err != nil {
		return nil, fmt.Errorf("qui: get torrent: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("qui: get torrent: status %d", resp.StatusCode())
	}

	if len(result.Torrents) == 0 {
		return nil, nil
	}
	return &result.Torrents[0], nil
}

// AddTag adds a tag to the given torrent hashes via the bulk-action endpoint.
func (c *Client) AddTag(hashes []string, tag string) error {
	payload := bulkActionPayload{
		Hashes:    hashes,
		Action:    "addTags",
		Tags:      tag,
		SelectAll: false,
	}

	resp, err := c.rc.R().
		SetBody(payload).
		Post(fmt.Sprintf("/api/instances/%d/torrents/bulk-action", c.instanceID))
	if err != nil {
		return fmt.Errorf("qui: add tag: %w", err)
	}
	if resp.IsError() {
		return fmt.Errorf("qui: add tag: status %d", resp.StatusCode())
	}
	return nil
}
