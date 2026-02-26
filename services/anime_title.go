package services

import (
	"context"
	"fmt"

	"github.com/ReinforceZwei/qb-auto/clients/animelist"
	tmdbclient "github.com/ReinforceZwei/qb-auto/clients/tmdb"
	"github.com/ReinforceZwei/qb-auto/llm"
)

// TitleResult holds the confirmed anime title and associated identifiers.
type TitleResult struct {
	// AnimeTitle is the Traditional Chinese (zh-TW) title confirmed via the anime list.
	AnimeTitle string
	// AnimeListID is the record ID in the anime list, used later to mark the anime as downloaded.
	AnimeListID int
	// TMDbID is the TMDb TV show ID of the matched entry.
	TMDbID int
}

// DetermineAnimeTitle resolves a downloaded torrent folder name to a confirmed
// Traditional Chinese anime title by:
//
//  1. Asking the LLM to extract the bare title from the folder name.
//  2. Searching TMDb for matching TV shows.
//  3. Asking the LLM to pick the best match from the results.
//  4. Retrieving the zh-TW title from TMDb for the chosen show.
//  5. Confirming the title exists in the anime list.
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
	// Step 1 — LLM: extract the bare anime title from the folder name.
	extractedTitle, err := llmClient.ExtractAnimeTitle(ctx, folderName)
	if err != nil {
		return nil, fmt.Errorf("determine anime title: %w", err)
	}

	// Step 2 — TMDb: search for TV shows matching the extracted title.
	candidates, err := tmdbClient.SearchAnime(extractedTitle)
	if err != nil {
		return nil, fmt.Errorf("determine anime title: tmdb search: %w", err)
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("determine anime title: no TMDb results for %q (folder: %q) — needs human review", extractedTitle, folderName)
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
		return nil, fmt.Errorf("determine anime title: %w", err)
	}
	if chosenIdx < 0 || chosenIdx >= len(candidates) {
		return nil, fmt.Errorf("determine anime title: LLM found no suitable TMDb match for %q — needs human review", folderName)
	}

	chosen := candidates[chosenIdx]

	// Step 4 — TMDb: fetch the Traditional Chinese (zh-TW) title.
	zhTitle, err := tmdbClient.GetTraditionalChineseTitle(chosen.ID)
	if err != nil {
		return nil, fmt.Errorf("determine anime title: get zh-TW title for TMDb id=%d: %w", chosen.ID, err)
	}

	// Step 5 — Anime list: confirm the title exists.
	records, err := animeListClient.Search(zhTitle)
	if err != nil {
		return nil, fmt.Errorf("determine anime title: anime list search: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("determine anime title: %q not found in anime list — needs human review", zhTitle)
	}

	return &TitleResult{
		AnimeTitle:  zhTitle,
		AnimeListID: records[0].ID,
		TMDbID:      chosen.ID,
	}, nil
}
