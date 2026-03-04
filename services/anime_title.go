package services

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ReinforceZwei/qb-auto/clients/animelist"
	braveclient "github.com/ReinforceZwei/qb-auto/clients/brave"
	tmdbclient "github.com/ReinforceZwei/qb-auto/clients/tmdb"
	wikiclient "github.com/ReinforceZwei/qb-auto/clients/wikipedia"
	"github.com/ReinforceZwei/qb-auto/llm"
)

// ResolveTitleResult holds the zh-TW anime title and TMDb ID resolved from a
// folder name without consulting the anime list.
type ResolveTitleResult struct {
	// AnimeTitle is the Traditional Chinese (zh-TW) title from TMDb or Wikipedia.
	AnimeTitle string
	// TMDbID is the TMDb TV show ID of the matched entry.
	TMDbID int
}

// TitleResult holds the confirmed anime title and associated identifiers.
type TitleResult struct {
	// AnimeTitle is the Traditional Chinese (zh-TW) title confirmed via the anime list.
	AnimeTitle string
	// AnimeListID is the record ID in the anime list, used later to mark the anime as downloaded.
	AnimeListID int
	// TMDbID is the TMDb TV show ID of the matched entry.
	TMDbID int
}

// ResolveAnimeTitle resolves a downloaded torrent folder name to a Traditional
// Chinese anime title by running steps 1–4 of the full determination flow:
//
//  1. Asking the LLM to extract the bare title from the folder name.
//  2. Searching TMDb for matching TV shows.
//  3. Asking the LLM to pick the best match from the results.
//  4. Retrieving the zh-TW title from TMDb for the chosen show.
//
// When braveClient and wikiClient are both non-nil, a Wikipedia-based fallback
// is attempted automatically if step 2 returns no candidates or step 3 finds no
// match (i.e. the primary TMDb path fails). The fallback:
//
//   - Searches Brave for "wikipedia <extractedTitle>" to locate a Wikipedia page.
//   - Parses the zh Wikipedia content to extract Chinese and original titles.
//   - Retries TMDb with the original title.
//   - Returns the zh-TW title from TMDb (case 8a) or the Chinese title from
//     Wikipedia when TMDb has no zh-TW translation (case 8b).
//
// The anime list is not consulted. Use DetermineAnimeTitle when the anime list
// confirmation step is also required.
func ResolveAnimeTitle(
	ctx context.Context,
	folderName string,
	llmClient *llm.Client,
	tmdbClient *tmdbclient.Client,
	braveClient *braveclient.Client,
	wikiClient *wikiclient.Client,
) (*ResolveTitleResult, error) {
	// Step 1 — LLM: extract the bare anime title from the folder name.
	extractedTitle, err := llmClient.ExtractAnimeTitle(ctx, folderName)
	if err != nil {
		return nil, fmt.Errorf("resolve anime title: %w", err)
	}

	// Step 2 — TMDb: search for TV shows matching the extracted title.
	candidates, err := tmdbClient.SearchAnime(extractedTitle)
	if err != nil {
		return nil, fmt.Errorf("resolve anime title: tmdb search: %w", err)
	}

	chosenIdx := -1
	if len(candidates) > 0 {
		// Step 3 — LLM: pick the best TMDb candidate.
		llmCandidates := make([]llm.TMDbCandidate, len(candidates))
		for i, c := range candidates {
			llmCandidates[i] = llm.TMDbCandidate{
				ID:           c.ID,
				Name:         c.Name,
				OriginalName: c.OriginalName,
				Overview:     c.Overview,
			}
		}
		chosenIdx, err = llmClient.PickBestTMDbMatch(ctx, folderName, extractedTitle, llmCandidates)
		if err != nil {
			return nil, fmt.Errorf("resolve anime title: %w", err)
		}
	}

	// Step 4 — If a TMDb candidate was chosen, try to fetch its zh-TW title.
	// A missing zh-TW translation is treated the same as no match: both fall
	// through to the Wikipedia fallback so it can supply a title from wikitext.
	if chosenIdx >= 0 && chosenIdx < len(candidates) {
		chosen := candidates[chosenIdx]
		zhTitle, err := tmdbClient.GetTraditionalChineseTitle(chosen.ID)
		if err == nil {
			return &ResolveTitleResult{AnimeTitle: zhTitle, TMDbID: chosen.ID}, nil
		}
		// No zh-TW title available for the matched show — fall through to Wikipedia.
	}

	// Wikipedia fallback: triggered when TMDb returned no candidates, the LLM
	// picked no match, or the chosen show has no zh-TW title.
	if braveClient != nil && wikiClient != nil {
		result, fallbackErr := resolveViaWikipedia(ctx, folderName, extractedTitle, llmClient, tmdbClient, braveClient, wikiClient)
		if fallbackErr != nil {
			return nil, fmt.Errorf("resolve anime title: TMDb path failed for %q, Wikipedia fallback also failed: %w", folderName, fallbackErr)
		}
		return result, nil
	}

	// No fallback available — report the most specific error.
	if len(candidates) == 0 {
		return nil, fmt.Errorf("resolve anime title: no TMDb results for %q (folder: %q) — needs human review", extractedTitle, folderName)
	}
	if chosenIdx < 0 || chosenIdx >= len(candidates) {
		return nil, fmt.Errorf("resolve anime title: LLM found no suitable TMDb match for %q — needs human review", folderName)
	}
	return nil, fmt.Errorf("resolve anime title: no zh-TW title on TMDb for %q (TMDb id=%d) — needs human review", extractedTitle, candidates[chosenIdx].ID)
}

// resolveViaWikipedia implements the Wikipedia-based fallback for title
// resolution. It is called when the primary TMDb search fails (no candidates or
// LLM picks no match). Steps correspond to the automated workflow in
// brave-search-wikipedia.md:
//
//  1. Brave-search "wikipedia <extractedTitle>" to get top web results.
//  2. Find the first Wikipedia URL in those results.
//  3. Parse the lang code and title from the URL.
//  4. If lang != "zh", fetch language links to find the zh equivalent title.
//  5. Fetch the zh Wikipedia page content (wikitext).
//  6. LLM extracts Chinese title, original title, and official TW translation.
//  7. Retry TMDb with the original title (or Chinese title as fallback search term).
//  8a. TMDb found + zh-TW title available → return TMDb title.
//  8b. TMDb found + no zh-TW title → return Wikipedia Chinese/official TW title.
//  8c. TMDb still found nothing → return error.
func resolveViaWikipedia(
	ctx context.Context,
	folderName string,
	extractedTitle string,
	llmClient *llm.Client,
	tmdbClient *tmdbclient.Client,
	braveClient *braveclient.Client,
	wikiClient *wikiclient.Client,
) (*ResolveTitleResult, error) {
	// Step 1 — Brave search: prepend "wikipedia" to boost Wikipedia results.
	query := "wikipedia " + extractedTitle
	searchResults, err := braveClient.Search(ctx, query, 10)
	if err != nil {
		return nil, fmt.Errorf("brave search: %w", err)
	}

	// Step 2 — Find the first Wikipedia URL.
	wikiURL := findWikipediaURL(searchResults)
	if wikiURL == "" {
		return nil, fmt.Errorf("no Wikipedia URL found in Brave search results for %q", extractedTitle)
	}

	// Step 3 — Parse lang and title from the Wikipedia URL.
	wikiLang, wikiTitle, err := parseWikipediaURL(wikiURL)
	if err != nil {
		return nil, fmt.Errorf("parse Wikipedia URL %q: %w", wikiURL, err)
	}

	// Step 4 — If not already a zh page, look up the zh language link.
	zhTitle := wikiTitle
	if wikiLang != "zh" {
		langLinks, err := wikiClient.GetLangLinks(ctx, wikiLang, wikiTitle)
		if err != nil {
			return nil, fmt.Errorf("get Wikipedia lang links for %q (%s): %w", wikiTitle, wikiLang, err)
		}
		zhLink := findLangLink(langLinks, "zh")
		if zhLink == nil {
			return nil, fmt.Errorf("no zh language link found for Wikipedia page %q (%s)", wikiTitle, wikiLang)
		}
		zhTitle = zhLink.Name
	}

	// Step 5 — Fetch zh Wikipedia page content (wikitext).
	wikitext, err := wikiClient.GetPageContent(ctx, "zh", zhTitle)
	if err != nil {
		return nil, fmt.Errorf("get Wikipedia page content for %q (zh): %w", zhTitle, err)
	}

	// Step 6 — LLM: extract Chinese title, original title, official TW translation.
	titleInfo, err := llmClient.ExtractTitleFromWikitext(ctx, wikitext)
	if err != nil {
		return nil, fmt.Errorf("extract title from wikitext: %w", err)
	}

	// Step 7 — Retry TMDb with the original title (preferred) or Chinese title.
	tmdbSearchTitle := titleInfo.OriginalTitle
	if tmdbSearchTitle == "" {
		tmdbSearchTitle = titleInfo.ChineseTitle
	}

	candidates, err := tmdbClient.SearchAnime(tmdbSearchTitle)
	if err != nil {
		return nil, fmt.Errorf("tmdb retry search with %q: %w", tmdbSearchTitle, err)
	}
	if len(candidates) == 0 {
		// 8c — even with Wikipedia data, TMDb has nothing.
		return nil, fmt.Errorf("no TMDb results for %q (from Wikipedia) — needs human review", tmdbSearchTitle)
	}

	llmCandidates := make([]llm.TMDbCandidate, len(candidates))
	for i, c := range candidates {
		llmCandidates[i] = llm.TMDbCandidate{
			ID:           c.ID,
			Name:         c.Name,
			OriginalName: c.OriginalName,
			Overview:     c.Overview,
		}
	}

	chosenIdx, err := llmClient.PickBestTMDbMatch(ctx, folderName, tmdbSearchTitle, llmCandidates)
	if err != nil {
		return nil, fmt.Errorf("llm pick best tmdb match (Wikipedia fallback): %w", err)
	}
	if chosenIdx < 0 || chosenIdx >= len(candidates) {
		// 8c — LLM still cannot identify a match.
		return nil, fmt.Errorf("LLM found no TMDb match for %q (Wikipedia fallback) — needs human review", tmdbSearchTitle)
	}

	chosen := candidates[chosenIdx]

	// Step 8a — Try to get the zh-TW title from TMDb.
	zhTWTitle, err := tmdbClient.GetTraditionalChineseTitle(chosen.ID)
	if err == nil {
		// 8a — TMDb has a zh-TW title: use it.
		return &ResolveTitleResult{AnimeTitle: zhTWTitle, TMDbID: chosen.ID}, nil
	}

	// 8b — No zh-TW title in TMDb: use the official TW translation from Wikipedia,
	// or fall back to the Chinese page title.
	animeTitle := titleInfo.OfficialTWTitle
	if animeTitle == "" {
		animeTitle = titleInfo.ChineseTitle
	}
	return &ResolveTitleResult{AnimeTitle: animeTitle, TMDbID: chosen.ID}, nil
}

// findWikipediaURL returns the first URL in results whose host contains
// "wikipedia.org", or an empty string if none is found.
func findWikipediaURL(results []braveclient.WebResult) string {
	for _, r := range results {
		if strings.Contains(r.URL, "wikipedia.org") {
			return r.URL
		}
	}
	return ""
}

// parseWikipediaURL extracts the language code and page title from a Wikipedia
// URL. It supports the two common URL patterns:
//   - https://zh.wikipedia.org/zh-tw/天久鷹央系列  → ("zh", "天久鷹央系列")
//   - https://en.wikipedia.org/wiki/A_Ninja_and_an_Assassin_Under_One_Roof → ("en", "A_Ninja_and_an_Assassin_Under_One_Roof")
func parseWikipediaURL(rawURL string) (lang, title string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}

	// Extract language from subdomain: "zh.wikipedia.org" → "zh"
	host := u.Hostname()
	parts := strings.SplitN(host, ".", 2)
	if len(parts) < 2 || parts[1] != "wikipedia.org" {
		return "", "", fmt.Errorf("not a Wikipedia URL: %q", rawURL)
	}
	lang = parts[0]

	// Extract the title from the last path segment.
	// Path examples:
	//   /zh-tw/天久鷹央系列         → segments ["", "zh-tw", "天久鷹央系列"]
	//   /wiki/A_Ninja_and_an_...   → segments ["", "wiki", "A_Ninja_..."]
	segments := strings.Split(u.Path, "/")
	// Filter empty segments (leading slash produces an empty first element).
	var nonEmpty []string
	for _, s := range segments {
		if s != "" {
			nonEmpty = append(nonEmpty, s)
		}
	}
	if len(nonEmpty) == 0 {
		return "", "", fmt.Errorf("Wikipedia URL has no path segments: %q", rawURL)
	}

	// URL-decode the title (e.g. %E5%A4%A9%E4%B9%85 → 天久).
	rawTitle := nonEmpty[len(nonEmpty)-1]
	decoded, decErr := url.PathUnescape(rawTitle)
	if decErr != nil {
		decoded = rawTitle
	}
	title = decoded
	return lang, title, nil
}

// findLangLink returns the LangLink for the given language code, or nil if not found.
func findLangLink(links []wikiclient.LangLink, lang string) *wikiclient.LangLink {
	for i := range links {
		if links[i].Lang == lang {
			return &links[i]
		}
	}
	return nil
}

// animeListChunkSize is the maximum number of anime list records sent to the LLM
// in a single call during the fallback search. Keeping it bounded avoids
// excessively long prompts when the watch list is large.
const animeListChunkSize = 150

// DetermineAnimeTitle resolves a downloaded torrent folder name to a confirmed
// Traditional Chinese anime title by running all 5 steps of the determination
// flow (steps 1–4 via ResolveAnimeTitle, then step 5):
//
//  5. Confirming the title exists in the anime list via LLM.
//
// Step 5 runs in two stages:
//   - Stage A: search the anime list by title; if results exist, ask the LLM to
//     pick the best match from those results.
//   - Stage B (fallback): if the search returned nothing or the LLM found no
//     match, fetch all unwatched+undownloaded records (sorted by addedTime desc)
//     and ask the LLM in chunks of animeListChunkSize until a match is found.
//
// Returns an error (stopping further processing) if no TMDb match is found,
// if the LLM cannot select a match, or if the anime list does not contain the
// resolved title. All such cases are left to human review.
func DetermineAnimeTitle(
	ctx context.Context,
	folderName string,
	llmClient *llm.Client,
	tmdbClient *tmdbclient.Client,
	animeListClient *animelist.Client,
	braveClient *braveclient.Client,
	wikiClient *wikiclient.Client,
) (*TitleResult, error) {
	resolved, err := ResolveAnimeTitle(ctx, folderName, llmClient, tmdbClient, braveClient, wikiClient)
	if err != nil {
		return nil, fmt.Errorf("determine anime title: %w", err)
	}

	// Stage A — search the anime list by the resolved title and ask the LLM to confirm.
	searchResults, err := animeListClient.Search(resolved.AnimeTitle)
	if err != nil {
		return nil, fmt.Errorf("determine anime title: anime list search: %w", err)
	}

	if len(searchResults) > 0 {
		candidates := toAnimeListCandidates(searchResults)
		idx, err := llmClient.PickBestAnimeListMatch(ctx, resolved.AnimeTitle, candidates)
		if err != nil {
			return nil, fmt.Errorf("determine anime title: llm pick from search results: %w", err)
		}
		if idx >= 0 && idx < len(searchResults) {
			matched := searchResults[idx]
			return &TitleResult{
				AnimeTitle:  resolved.AnimeTitle,
				AnimeListID: matched.ID,
				TMDbID:      resolved.TMDbID,
			}, nil
		}
	}

	// Stage B — fallback: iterate the full unwatched+undownloaded list in chunks.
	fullList, err := animeListClient.GetUnwatchedUndownloaded()
	if err != nil {
		return nil, fmt.Errorf("determine anime title: get unwatched undownloaded: %w", err)
	}

	for start := 0; start < len(fullList); start += animeListChunkSize {
		end := min(start+animeListChunkSize, len(fullList))
		chunk := fullList[start:end]

		candidates := toAnimeListCandidates(chunk)
		idx, err := llmClient.PickBestAnimeListMatch(ctx, resolved.AnimeTitle, candidates)
		if err != nil {
			return nil, fmt.Errorf("determine anime title: llm pick from fallback chunk [%d:%d]: %w", start, end, err)
		}
		if idx >= 0 && idx < len(chunk) {
			matched := chunk[idx]
			return &TitleResult{
				AnimeTitle:  resolved.AnimeTitle,
				AnimeListID: matched.ID,
				TMDbID:      resolved.TMDbID,
			}, nil
		}
	}

	return nil, fmt.Errorf("determine anime title: %q not found in anime list — needs human review", resolved.AnimeTitle)
}

// toAnimeListCandidates converts a slice of AnimeRecord to the slim candidate
// type that is passed to the LLM (ID and Name only).
func toAnimeListCandidates(records []animelist.AnimeRecord) []llm.AnimeListCandidate {
	out := make([]llm.AnimeListCandidate, len(records))
	for i, r := range records {
		out[i] = llm.AnimeListCandidate{ID: r.ID, Name: r.Name}
	}
	return out
}
