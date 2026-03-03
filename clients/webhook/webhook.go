package webhook

import (
	"fmt"

	"github.com/ReinforceZwei/qb-auto/config"
	"resty.dev/v3"
)

type discordFooter struct {
	Text string `json:"text"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordEmbed struct {
	Title  string         `json:"title"`
	Color  int            `json:"color"`
	Fields []discordField `json:"fields"`
	Footer discordFooter  `json:"footer"`
}

type discordPayload struct {
	Username string         `json:"username"`
	Embeds   []discordEmbed `json:"embeds"`
}

// Client sends Discord webhook notifications.
type Client struct {
	webhookURL string
	rc         *resty.Client
}

// New creates a Client from cfg. If WebhookURL is empty, Send is a no-op.
func New(cfg *config.Config) *Client {
	rc := resty.New()
	return &Client{webhookURL: cfg.WebhookURL, rc: rc}
}

func (c *Client) post(payload discordPayload) error {
	resp, err := c.rc.R().
		SetBody(payload).
		Post(c.webhookURL)
	if err != nil {
		return fmt.Errorf("webhook: send: %w", err)
	}
	if resp.IsError() {
		return fmt.Errorf("webhook: send: status %d", resp.StatusCode())
	}
	return nil
}

// Send posts a Discord embed notification for a completed torrent.
// If the client has no webhook URL configured, Send returns nil immediately.
func (c *Client) Send(torrentName, category string) error {
	if c.webhookURL == "" {
		return nil
	}

	if category == "" {
		category = "None"
	}

	payload := discordPayload{
		Username: "qbittorrent",
		Embeds: []discordEmbed{
			{
				Title: "Done: " + torrentName,
				Color: 3447003,
				Fields: []discordField{
					{Name: "Name", Value: torrentName, Inline: false},
					{Name: "Category", Value: category, Inline: true},
				},
			Footer: discordFooter{Text: "qBittorrent"},
		},
	},
	}

	return c.post(payload)
}

// SendError posts a Discord embed notification for a failed job.
// If the client has no webhook URL configured, SendError returns nil immediately.
func (c *Client) SendError(torrentName, category, errMsg string) error {
	if c.webhookURL == "" {
		return nil
	}

	if category == "" {
		category = "None"
	}
	if torrentName == "" {
		torrentName = "Unknown"
	}

	payload := discordPayload{
		Username: "qbittorrent",
		Embeds: []discordEmbed{
			{
				Title: "Failed: " + torrentName,
				Color: 15158332,
				Fields: []discordField{
					{Name: "Name", Value: torrentName, Inline: false},
					{Name: "Category", Value: category, Inline: true},
					{Name: "Error", Value: errMsg, Inline: false},
				},
				Footer: discordFooter{Text: "qBittorrent"},
			},
		},
	}

	return c.post(payload)
}
