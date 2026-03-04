package webhook

import (
	"fmt"

	"github.com/ReinforceZwei/qb-auto/config"
	"resty.dev/v3"
)

// DiscordFooter is the footer section of a Discord embed.
type DiscordFooter struct {
	Text string `json:"text"`
}

// DiscordField is a single field within a Discord embed.
type DiscordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// DiscordEmbed represents a Discord message embed.
type DiscordEmbed struct {
	Title  string         `json:"title"`
	Color  int            `json:"color"`
	Fields []DiscordField `json:"fields"`
	Footer DiscordFooter  `json:"footer"`
}

type discordPayload struct {
	Username string         `json:"username"`
	Embeds   []DiscordEmbed `json:"embeds"`
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

// Send posts a Discord embed notification.
// If the client has no webhook URL configured, Send returns nil immediately.
func (c *Client) Send(embed DiscordEmbed) error {
	if c.webhookURL == "" {
		return nil
	}

	payload := discordPayload{
		Username: "qbittorrent",
		Embeds:   []DiscordEmbed{embed},
	}

	return c.post(payload)
}
