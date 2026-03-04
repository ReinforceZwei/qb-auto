package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ReinforceZwei/qb-auto/config"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// TMDbCandidate holds the TMDb TV show information passed to the LLM for match selection.
type TMDbCandidate struct {
	ID           int
	Name         string
	OriginalName string
	Overview     string
}

// AnimeListCandidate holds the anime list record information passed to the LLM for match selection.
// Only ID and Name are included as specified — the list can be large and extra fields add noise.
type AnimeListCandidate struct {
	ID   int
	Name string
}

// Client wraps an eino BaseChatModel and provides higher-level anime title helpers.
type Client struct {
	model model.BaseChatModel
}

// New creates a Client using an OpenAI-compatible endpoint configured via cfg.
func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  cfg.LLMAPIKey,
		Model:   cfg.LLMModelName,
		BaseURL: cfg.LLMBaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("llm: init chat model: %w", err)
	}
	return &Client{model: chatModel}, nil
}

// ExtractAnimeTitle asks the LLM to strip metadata from a torrent folder name and
// return only the core anime title.
func (c *Client) ExtractAnimeTitle(ctx context.Context, folderName string) (string, error) {
	messages := []*schema.Message{
		{Role: schema.System, Content: promptExtractTitle},
		{Role: schema.User, Content: folderName},
	}

	resp, err := c.model.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("llm: extract anime title: %w", err)
	}

	var result struct {
		Title string `json:"title"`
	}
	if err := parseJSONResponse(resp.Content, &result); err != nil {
		return "", fmt.Errorf("llm: extract anime title: %w", err)
	}
	if result.Title == "" {
		return "", fmt.Errorf("llm: extract anime title: LLM returned empty title")
	}
	return result.Title, nil
}

// WikipediaTitleInfo holds the anime title fields extracted from a Wikipedia
// wikitext blob by the LLM.
type WikipediaTitleInfo struct {
	// ChineseTitle is the Traditional Chinese page title from Wikipedia.
	ChineseTitle string
	// OriginalTitle is the original Japanese title from the infobox.
	OriginalTitle string
	// OfficialTWTitle is the official Taiwan translation from the 正式譯名 infobox
	// field, or an empty string when the field is absent.
	OfficialTWTitle string
}

// ExtractTitleFromWikitext asks the LLM to extract anime title information from
// a Traditional Chinese Wikipedia wikitext blob. It returns the Chinese title,
// the original Japanese title, and the official TW translation (if any).
func (c *Client) ExtractTitleFromWikitext(ctx context.Context, wikitext string) (*WikipediaTitleInfo, error) {
	messages := []*schema.Message{
		{Role: schema.System, Content: promptExtractTitleFromWikitext},
		{Role: schema.User, Content: wikitext},
	}

	resp, err := c.model.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("llm: extract title from wikitext: %w", err)
	}

	var result struct {
		ChineseTitle    string `json:"chinese_title"`
		OriginalTitle   string `json:"original_title"`
		OfficialTWTitle string `json:"official_tw_title"`
	}
	if err := parseJSONResponse(resp.Content, &result); err != nil {
		return nil, fmt.Errorf("llm: extract title from wikitext: %w", err)
	}
	if result.ChineseTitle == "" {
		return nil, fmt.Errorf("llm: extract title from wikitext: LLM returned empty chinese_title")
	}
	return &WikipediaTitleInfo{
		ChineseTitle:    result.ChineseTitle,
		OriginalTitle:   result.OriginalTitle,
		OfficialTWTitle: result.OfficialTWTitle,
	}, nil
}

// PickBestTMDbMatch asks the LLM to choose the TMDb result that best matches the
// original folder name. Returns the 0-based index of the chosen candidate, or -1
// if the LLM finds no suitable match.
func (c *Client) PickBestTMDbMatch(ctx context.Context, folderName, extractedTitle string, candidates []TMDbCandidate) (int, error) {
	userMsg := buildPickMatchUserMessage(folderName, extractedTitle, candidates)

	messages := []*schema.Message{
		{Role: schema.System, Content: promptPickBestMatch},
		{Role: schema.User, Content: userMsg},
	}

	resp, err := c.model.Generate(ctx, messages)
	if err != nil {
		return -1, fmt.Errorf("llm: pick best tmdb match: %w", err)
	}

	var result struct {
		Index int `json:"index"`
	}
	if err := parseJSONResponse(resp.Content, &result); err != nil {
		return -1, fmt.Errorf("llm: pick best tmdb match: %w", err)
	}
	return result.Index, nil
}

// PickBestAnimeListMatch asks the LLM to choose the anime list record that best
// matches the resolved Traditional Chinese title. Returns the 0-based index of
// the chosen candidate, or -1 if the LLM finds no suitable match.
func (c *Client) PickBestAnimeListMatch(ctx context.Context, animeTitle string, candidates []AnimeListCandidate) (int, error) {
	userMsg := buildAnimeListMatchUserMessage(animeTitle, candidates)

	messages := []*schema.Message{
		{Role: schema.System, Content: promptPickBestAnimeListMatch},
		{Role: schema.User, Content: userMsg},
	}

	resp, err := c.model.Generate(ctx, messages)
	if err != nil {
		return -1, fmt.Errorf("llm: pick best anime list match: %w", err)
	}

	var result struct {
		Index int `json:"index"`
	}
	if err := parseJSONResponse(resp.Content, &result); err != nil {
		return -1, fmt.Errorf("llm: pick best anime list match: %w", err)
	}
	return result.Index, nil
}

// buildAnimeListMatchUserMessage formats the user message listing all anime list candidates.
func buildAnimeListMatchUserMessage(animeTitle string, candidates []AnimeListCandidate) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Resolved anime title: %s\n\n", animeTitle)
	sb.WriteString("Anime list records:\n")
	for i, c := range candidates {
		fmt.Fprintf(&sb, "[%d] name=%s\n", i, c.Name)
	}
	return sb.String()
}

// buildPickMatchUserMessage formats the user message listing all TMDb candidates.
func buildPickMatchUserMessage(folderName, extractedTitle string, candidates []TMDbCandidate) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Original folder name: %s\n", folderName)
	fmt.Fprintf(&sb, "Extracted title: %s\n\n", extractedTitle)
	sb.WriteString("TMDb search results:\n")
	for i, c := range candidates {
		fmt.Fprintf(&sb, "[%d] Name: %s | Original name: %s | Overview: %s\n",
			i, c.Name, c.OriginalName, c.Overview)
	}
	return sb.String()
}

// parseJSONResponse strips optional markdown code fences and unmarshals JSON.
func parseJSONResponse(raw string, v any) error {
	s := strings.TrimSpace(raw)
	// Strip ```json ... ``` or ``` ... ``` fences that some models add
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	if err := json.Unmarshal([]byte(s), v); err != nil {
		return fmt.Errorf("parse JSON response: %w (raw: %q)", err, raw)
	}
	return nil
}
