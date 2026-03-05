package workers

import (
	"context"
	"path"
	"strconv"

	"github.com/pocketbase/dbx"

	"github.com/ReinforceZwei/qb-auto/clients/animelist"
	rsyncclient "github.com/ReinforceZwei/qb-auto/clients/rsync"
	quiclient "github.com/ReinforceZwei/qb-auto/clients/qui"
	"github.com/ReinforceZwei/qb-auto/config"
	"github.com/ReinforceZwei/qb-auto/models"
	"github.com/pocketbase/pocketbase/core"
)

// RsyncWorker processes pending_rsync jobs with a single goroutine. It copies
// torrent files to the NAS via rsync, marks the torrent done in qui, marks the
// anime as downloaded in the anime list, and transitions the job to
// pending_notify.
type RsyncWorker struct {
	app             core.App
	cfg             *config.Config
	quiClient       *quiclient.Client
	rsyncClient     *rsyncclient.Client
	animeListClient *animelist.Client
	jobCh           chan string // buffered channel of job record IDs
}

// NewRsyncWorker creates a RsyncWorker. The job channel is buffered at 64 to
// absorb bursts without blocking hook handlers.
func NewRsyncWorker(
	app core.App,
	cfg *config.Config,
	quiClient *quiclient.Client,
	rsyncClient *rsyncclient.Client,
	animeListClient *animelist.Client,
) *RsyncWorker {
	return &RsyncWorker{
		app:             app,
		cfg:             cfg,
		quiClient:       quiClient,
		rsyncClient:     rsyncClient,
		animeListClient: animeListClient,
		jobCh:           make(chan string, 64),
	}
}

// Register attaches PocketBase hooks so that any jobs record created or updated
// with status="pending_rsync" is dispatched to the worker.
func (w *RsyncWorker) Register() {
	dispatch := func(e *core.RecordEvent) error {
		record := e.Record
		if record.GetString("status") == models.JobStatusPendingRsync {
			select {
			case w.jobCh <- record.Id:
			default:
				w.app.Logger().Warn("rsync_worker: job queue full, dropping job", "jobId", record.Id)
			}
		}
		return e.Next()
	}

	w.app.OnRecordAfterCreateSuccess("jobs").BindFunc(dispatch)
	w.app.OnRecordAfterUpdateSuccess("jobs").BindFunc(dispatch)
}

// Start spawns a single goroutine that consumes from the job channel until ctx
// is cancelled. rsync is I/O-bound and serialised to avoid saturating the NAS
// link with concurrent transfers.
func (w *RsyncWorker) Start(ctx context.Context) {
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
	w.app.Logger().Info("rsync_worker: started")
}

// processJob fetches the record, verifies it is still eligible, then runs the
// full rsync transfer flow.
func (w *RsyncWorker) processJob(ctx context.Context, recordID string) {
	// Fetch a fresh copy to avoid acting on stale hook data.
	record, err := w.app.FindRecordById("jobs", recordID)
	if err != nil {
		w.app.Logger().Error("rsync_worker: fetch record failed", "jobId", recordID, "error", err)
		return
	}

	// Re-check eligibility — guard against duplicate dispatches.
	if record.GetString("status") != models.JobStatusPendingRsync {
		return
	}

	// Claim the job.
	record.Set("status", models.JobStatusProcessingRsync)
	if err := w.app.Save(record); err != nil {
		w.app.Logger().Error("rsync_worker: claim job failed", "jobId", recordID, "error", err)
		return
	}

	torrentHash := record.GetString("torrent_hash")
	animeTitle := record.GetString("anime_title")
	animeListIDStr := record.GetString("anime_list_id")
	tmdbID := record.GetInt("tmdb_id")

	// Resolve the local content path from qui.
	torrent, err := w.quiClient.GetTorrent(torrentHash)
	if err != nil {
		w.failJob(record, "get torrent info: "+err.Error())
		return
	}
	if torrent == nil {
		w.failJob(record, "torrent not found in qui for hash "+torrentHash)
		return
	}

	// Build dest: <NAS_ANIME_BASE_PATH>/<animeTitle>
	// The rsync module maps to the NAS share root (like a drive letter), so
	// the full NAS path becomes: <module>/<NASAnimeBasePath>/<animeTitle>/<folder>.
	dest := path.Join(w.cfg.NASAnimeBasePath, animeTitle)

	w.app.Logger().Debug("rsync_worker: starting transfer",
		"jobId", recordID,
		"src", torrent.ContentPath,
		"dest", dest,
		"animeTitle", animeTitle,
	)

	if err := w.rsyncClient.Copy(ctx, torrent.ContentPath, dest); err != nil {
		w.failJob(record, "rsync copy: "+err.Error())
		return
	}

	w.app.Logger().Debug("rsync_worker: transfer complete", "jobId", recordID)

	// Mark the torrent as done in qBittorrent via qui.
	if err := w.quiClient.AddTag([]string{torrentHash}, "done"); err != nil {
		w.failJob(record, "add 'done' tag: "+err.Error())
		return
	}

	// Mark the anime as downloaded in the anime list.
	animeListID, err := strconv.Atoi(animeListIDStr)
	if err != nil {
		w.failJob(record, "invalid anime_list_id "+strconv.Quote(animeListIDStr)+": "+err.Error())
		return
	}
	if err := w.animeListClient.MarkDownloaded(animeListID, tmdbID); err != nil {
		w.failJob(record, "mark downloaded in anime list: "+err.Error())
		return
	}

	record.Set("status", models.JobStatusPendingNotify)
	if err := w.app.Save(record); err != nil {
		w.app.Logger().Error("rsync_worker: save final status failed", "jobId", recordID, "error", err)
		return
	}

	w.app.Logger().Info("rsync_worker: job completed", "jobId", recordID, "animeTitle", animeTitle)
}

// Recover re-enqueues any jobs that were left in a non-terminal state from a
// previous run. Jobs stuck in processing_rsync are rolled back to pending_rsync
// so they re-enter the normal flow cleanly. rsync is idempotent, so re-running
// a partially completed transfer is safe.
func (w *RsyncWorker) Recover() {
	records, err := w.app.FindRecordsByFilter(
		"jobs",
		"status = {:pending} || status = {:processing}",
		"", 0, 0,
		dbx.Params{
			"pending":    models.JobStatusPendingRsync,
			"processing": models.JobStatusProcessingRsync,
		},
	)
	if err != nil {
		w.app.Logger().Error("rsync_worker: recovery query failed", "error", err)
		return
	}
	if len(records) == 0 {
		return
	}

	w.app.Logger().Info("rsync_worker: recovering stuck jobs", "count", len(records))
	for _, record := range records {
		if record.GetString("status") == models.JobStatusProcessingRsync {
			record.Set("status", models.JobStatusPendingRsync)
			if err := w.app.Save(record); err != nil {
				w.app.Logger().Error("rsync_worker: recover rollback failed", "jobId", record.Id, "error", err)
				continue
			}
		}
		select {
		case w.jobCh <- record.Id:
		default:
			w.app.Logger().Warn("rsync_worker: job queue full during recovery, dropping job", "jobId", record.Id)
		}
	}
}

// failJob transitions a job record to the error state with a message.
func (w *RsyncWorker) failJob(record *core.Record, msg string) {
	w.app.Logger().Error("rsync_worker: job failed", "jobId", record.Id, "error", msg)
	record.Set("status", models.JobStatusError)
	record.Set("error", msg)
	if err := w.app.Save(record); err != nil {
		w.app.Logger().Error("rsync_worker: save error state failed", "jobId", record.Id, "error", err)
	}
}
