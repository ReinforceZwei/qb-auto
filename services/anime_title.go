package services

import (
	"context"
	"fmt"

	"github.com/ReinforceZwei/qb-auto/clients/animelist"
	tmdbclient "github.com/ReinforceZwei/qb-auto/clients/tmdb"
	"github.com/ReinforceZwei/qb-auto/llm"
)

// ResolveTitleResult holds the zh-TW anime title and TMDb ID resolved from a
// folder name without consulting the anime list.
type ResolveTitleResult struct {
	// AnimeTitle is the Traditional Chinese (zh-TW) title from TMDb.
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
// The anime list is not consulted. Use DetermineAnimeTitle when the anime list
// confirmation step is also required.
func ResolveAnimeTitle(
	ctx context.Context,
	folderName string,
	llmClient *llm.Client,
	tmdbClient *tmdbclient.Client,
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
	if len(candidates) == 0 {
		return nil, fmt.Errorf("resolve anime title: no TMDb results for %q (folder: %q) — needs human review", extractedTitle, folderName)
	}

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

	chosenIdx, err := llmClient.PickBestTMDbMatch(ctx, folderName, extractedTitle, llmCandidates)
	if err != nil {
		return nil, fmt.Errorf("resolve anime title: %w", err)
	}
	if chosenIdx < 0 || chosenIdx >= len(candidates) {
		return nil, fmt.Errorf("resolve anime title: LLM found no suitable TMDb match for %q — needs human review", folderName)
	}

	chosen := candidates[chosenIdx]

	// Step 4 — TMDb: fetch the Traditional Chinese (zh-TW) title.
	zhTitle, err := tmdbClient.GetTraditionalChineseTitle(chosen.ID)
	if err != nil {
		return nil, fmt.Errorf("resolve anime title: get zh-TW title for TMDb id=%d: %w", chosen.ID, err)
	}

	return &ResolveTitleResult{
		AnimeTitle: zhTitle,
		TMDbID:     chosen.ID,
	}, nil
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
) (*TitleResult, error) {
	resolved, err := ResolveAnimeTitle(ctx, folderName, llmClient, tmdbClient)
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
