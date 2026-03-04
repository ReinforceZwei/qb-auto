package workers

import (
	"context"

	"github.com/ReinforceZwei/qb-auto/clients/animelist"
	braveclient "github.com/ReinforceZwei/qb-auto/clients/brave"
	quiclient "github.com/ReinforceZwei/qb-auto/clients/qui"
	tmdbclient "github.com/ReinforceZwei/qb-auto/clients/tmdb"
	wikiclient "github.com/ReinforceZwei/qb-auto/clients/wikipedia"
	"github.com/ReinforceZwei/qb-auto/config"
	"github.com/ReinforceZwei/qb-auto/llm"
	"github.com/ReinforceZwei/qb-auto/models"
	"github.com/ReinforceZwei/qb-auto/services"
	"github.com/pocketbase/pocketbase/core"
)

// TitleWorker processes pending jobs to determine the anime title using a pool
// of goroutines. The pool size is controlled by cfg.TitleWorkerCount.
type TitleWorker struct {
	app             core.App
	cfg             *config.Config
	quiClient       *quiclient.Client
	llmClient       *llm.Client
	tmdbClient      *tmdbclient.Client
	animeListClient *animelist.Client
	// braveClient and wikiClient are optional; when non-nil they enable the
	// Wikipedia-based fallback for title resolution.
	braveClient *braveclient.Client
	wikiClient  *wikiclient.Client
	jobCh       chan string // buffered channel of job record IDs
}

// NewTitleWorker creates a TitleWorker. The job channel is buffered at
// TitleWorkerCount*10 to absorb bursts without blocking hook handlers.
// braveClient and wikiClient may be nil; pass them to enable the Wikipedia
// fallback when TMDb finds no match.
func NewTitleWorker(
	app core.App,
	cfg *config.Config,
	quiClient *quiclient.Client,
	llmClient *llm.Client,
	tmdbClient *tmdbclient.Client,
	animeListClient *animelist.Client,
	braveClient *braveclient.Client,
	wikiClient *wikiclient.Client,
) *TitleWorker {
	return &TitleWorker{
		app:             app,
		cfg:             cfg,
		quiClient:       quiClient,
		llmClient:       llmClient,
		tmdbClient:      tmdbClient,
		animeListClient: animeListClient,
		braveClient:     braveClient,
		wikiClient:      wikiClient,
		jobCh:           make(chan string, cfg.TitleWorkerCount*10),
	}
}

// Register attaches PocketBase hooks so that any jobs record that is created or
// updated with status="pending" and an empty anime_title is dispatched to the
// worker pool.
func (w *TitleWorker) Register() {
	dispatch := func(e *core.RecordEvent) error {
		record := e.Record
		if record.GetString("status") == models.JobStatusPending &&
			record.GetString("anime_title") == "" &&
			record.GetString("category") == "anime" {
			select {
			case w.jobCh <- record.Id:
			default:
				w.app.Logger().Warn("title_worker: job queue full, dropping job", "jobId", record.Id)
			}
		}
		return e.Next()
	}

	w.app.OnRecordAfterCreateSuccess("jobs").BindFunc(dispatch)
	w.app.OnRecordAfterUpdateSuccess("jobs").BindFunc(dispatch)
}

// Start spawns TitleWorkerCount goroutines that each consume from the job
// channel until ctx is cancelled.
func (w *TitleWorker) Start(ctx context.Context) {
	for range w.cfg.TitleWorkerCount {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case id := <-w.jobCh:
					w.processJob(ctx, id)
				}
			}
		}()
	}
	w.app.Logger().Info("title_worker: started", "workers", w.cfg.TitleWorkerCount)
}

// processJob fetches the record, verifies it is still eligible, then runs the
// full anime title determination flow, updating the record with the result.
func (w *TitleWorker) processJob(ctx context.Context, recordID string) {
	// Fetch fresh copy of the record to avoid acting on stale hook data.
	record, err := w.app.FindRecordById("jobs", recordID)
	if err != nil {
		w.app.Logger().Error("title_worker: fetch record failed", "jobId", recordID, "error", err)
		return
	}

	// Re-check eligibility — a concurrent worker may have claimed it already.
	if record.GetString("status") != models.JobStatusPending ||
		record.GetString("anime_title") != "" {
		return
	}

	// Claim the job by transitioning to processing_title.
	record.Set("status", models.JobStatusProcessingTitle)
	if err := w.app.Save(record); err != nil {
		w.app.Logger().Error("title_worker: claim job failed", "jobId", recordID, "error", err)
		return
	}

	torrentHash := record.GetString("torrent_hash")

	torrent, err := w.quiClient.GetTorrent(torrentHash)
	if err != nil {
		w.failJob(record, "get torrent info: "+err.Error())
		return
	}
	if torrent == nil {
		w.failJob(record, "torrent not found in qui for hash "+torrentHash)
		return
	}

	w.app.Logger().Debug("title_worker: fetched torrent info", "jobId", recordID, "torrentName", torrent.Name, "torrentHash", torrentHash)

	result, err := services.DetermineAnimeTitle(ctx, torrent.Name, w.llmClient, w.tmdbClient, w.animeListClient, w.braveClient, w.wikiClient)
	if err != nil {
		w.failJob(record, err.Error())
		return
	}

	w.app.Logger().Debug("title_worker: title determined", "jobId", recordID, "animeTitle", result.AnimeTitle, "tmdbId", result.TMDbID, "animeListId", result.AnimeListID)

	record.Set("anime_title", result.AnimeTitle)
	record.Set("anime_list_id", result.AnimeListID)
	record.Set("tmdb_id", result.TMDbID)
	record.Set("status", models.JobStatusPendingRsync)
	if err := w.app.Save(record); err != nil {
		w.app.Logger().Error("title_worker: save result failed", "jobId", recordID, "error", err)
		return
	}

	w.app.Logger().Info("title_worker: job completed", "jobId", recordID, "animeTitle", result.AnimeTitle)
}

// failJob transitions a job record to the error state with a message.
func (w *TitleWorker) failJob(record *core.Record, msg string) {
	w.app.Logger().Error("title_worker: job failed", "jobId", record.Id, "error", msg)
	record.Set("status", models.JobStatusError)
	record.Set("error", msg)
	if err := w.app.Save(record); err != nil {
		w.app.Logger().Error("title_worker: save error state failed", "jobId", record.Id, "error", err)
	}
}
