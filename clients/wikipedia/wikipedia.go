package wikipedia

import (
	"context"
	"fmt"

	"resty.dev/v3"
)

// LangLink represents a single language link returned by the Wikipedia API.
type LangLink struct {
	// Lang is the BCP-47 language code (e.g. "zh", "en", "ja").
	Lang string
	// URL is the full Wikipedia URL for this language variant.
	URL string
	// Name is the page title in the target language.
	Name string
}

// Client wraps the public Wikipedia Action API.
// No authentication is required.
type Client struct {
	rc *resty.Client
}

// New creates a Wikipedia API Client.
func New() *Client {
	rc := resty.New().
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", "qb-auto/1.0 (github.com/ReinforceZwei/qb-auto)")
	return &Client{rc: rc}
}

// apiBase returns the base URL for the Wikipedia Action API for the given language.
func apiBase(lang string) string {
	return fmt.Sprintf("https://%s.wikipedia.org/w/api.php", lang)
}

// langLinksResponse is the minimal structure we need from the langlinks API response.
type langLinksResponse struct {
	Query struct {
		Pages map[string]struct {
			Title     string `json:"title"`
			LangLinks []struct {
				Lang string `json:"lang"`
				URL  string `json:"url"`
				Name string `json:"*"`
			} `json:"langlinks"`
		} `json:"pages"`
	} `json:"query"`
}

// GetLangLinks returns the language links for the Wikipedia page identified by
// lang and title. lang is the source language code (e.g. "en"), title is the
// page title (may be URL-encoded or plain text).
func (c *Client) GetLangLinks(ctx context.Context, lang, title string) ([]LangLink, error) {
	var resp langLinksResponse
	httpResp, err := c.rc.R().
		SetContext(ctx).
		SetQueryParam("action", "query").
		SetQueryParam("prop", "langlinks").
		SetQueryParam("titles", title).
		SetQueryParam("lllimit", "500").
		SetQueryParam("llprop", "url").
		SetQueryParam("format", "json").
		SetResult(&resp).
		Get(apiBase(lang))
	if err != nil {
		return nil, fmt.Errorf("wikipedia: get langlinks for %q (%s): %w", title, lang, err)
	}
	if httpResp.IsError() {
		return nil, fmt.Errorf("wikipedia: get langlinks for %q (%s): HTTP %d", title, lang, httpResp.StatusCode())
	}

	var links []LangLink
	for _, page := range resp.Query.Pages {
		for _, ll := range page.LangLinks {
			links = append(links, LangLink{
				Lang: ll.Lang,
				URL:  ll.URL,
				Name: ll.Name,
			})
		}
	}
	return links, nil
}

// pageContentResponse is the minimal structure we need from the revisions API response.
type pageContentResponse struct {
	Query struct {
		Pages map[string]struct {
			Title     string `json:"title"`
			Revisions []struct {
				Slots struct {
					Main struct {
						Content string `json:"*"`
					} `json:"main"`
				} `json:"slots"`
			} `json:"revisions"`
		} `json:"pages"`
	} `json:"query"`
}

// GetPageContent fetches the wikitext content of the Wikipedia page identified
// by lang and title. lang is the language code (e.g. "zh"), title is the page
// title. Returns the raw wikitext string.
func (c *Client) GetPageContent(ctx context.Context, lang, title string) (string, error) {
	var resp pageContentResponse
	httpResp, err := c.rc.R().
		SetContext(ctx).
		SetQueryParam("action", "query").
		SetQueryParam("prop", "revisions").
		SetQueryParam("titles", title).
		SetQueryParam("rvprop", "content").
		SetQueryParam("rvslots", "main").
		SetQueryParam("format", "json").
		SetResult(&resp).
		Get(apiBase(lang))
	if err != nil {
		return "", fmt.Errorf("wikipedia: get page content for %q (%s): %w", title, lang, err)
	}
	if httpResp.IsError() {
		return "", fmt.Errorf("wikipedia: get page content for %q (%s): HTTP %d", title, lang, httpResp.StatusCode())
	}

	for _, page := range resp.Query.Pages {
		if len(page.Revisions) > 0 {
			content := page.Revisions[0].Slots.Main.Content
			if content == "" {
				return "", fmt.Errorf("wikipedia: page %q (%s) has empty content", title, lang)
			}
			return content, nil
		}
	}
	return "", fmt.Errorf("wikipedia: page %q (%s) not found or has no revisions", title, lang)
}
