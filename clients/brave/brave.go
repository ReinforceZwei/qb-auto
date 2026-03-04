package brave

import (
	"context"
	"fmt"

	"resty.dev/v3"
)

const baseURL = "https://api.search.brave.com/res/v1"

// WebResult holds the fields from a Brave web search result that are relevant
// to finding Wikipedia pages.
type WebResult struct {
	Title       string
	URL         string
	Description string
}

// Client is a thin wrapper around the Brave Web Search API.
type Client struct {
	rc *resty.Client
}

// New creates a Client authenticated with the provided API key.
func New(apiKey string) *Client {
	rc := resty.New().
		SetBaseURL(baseURL).
		SetHeader("Accept", "application/json").
		SetHeader("Accept-Encoding", "gzip").
		SetHeader("X-Subscription-Token", apiKey)
	return &Client{rc: rc}
}

// braveWebResponse is the minimal subset of the Brave Search API response that
// we decode. Only web.results are extracted; all other top-level fields are
// ignored.
type braveWebResponse struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
		} `json:"results"`
	} `json:"web"`
}

// Search performs a Brave web search and returns up to count results (max 20).
// Pass count ≤ 0 to use the API default (20).
func (c *Client) Search(ctx context.Context, query string, count int) ([]WebResult, error) {
	req := c.rc.R().
		SetContext(ctx).
		SetQueryParam("q", query).
		SetQueryParam("result_filter", "web")

	if count > 0 {
		req.SetQueryParam("count", fmt.Sprintf("%d", count))
	}

	var resp braveWebResponse
	httpResp, err := req.SetResult(&resp).Get("/web/search")
	if err != nil {
		return nil, fmt.Errorf("brave: search %q: %w", query, err)
	}
	if httpResp.IsError() {
		return nil, fmt.Errorf("brave: search %q: HTTP %d: %s", query, httpResp.StatusCode(), httpResp.String())
	}

	results := make([]WebResult, 0, len(resp.Web.Results))
	for _, r := range resp.Web.Results {
		results = append(results, WebResult{
			Title:       r.Title,
			URL:         r.URL,
			Description: r.Description,
		})
	}
	return results, nil
}
