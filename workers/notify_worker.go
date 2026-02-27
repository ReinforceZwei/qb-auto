package workers

import (
	"context"
	"time"

	webhookclient "github.com/ReinforceZwei/qb-auto/clients/webhook"
	quiclient "github.com/ReinforceZwei/qb-auto/clients/qui"
	"github.com/ReinforceZwei/qb-auto/config"
	"github.com/ReinforceZwei/qb-auto/models"
	"github.com/pocketbase/pocketbase/core"
)

// NotifyWorker picks up pending_notify jobs, sends a Discord webhook embed, and
// transitions jobs to done. A single goroutine is used; notifications are fast
// and do not need parallelism.
type NotifyWorker struct {
	app           core.App
	cfg           *config.Config
	quiClient     *quiclient.Client
	webhookClient *webhookclient.Client
	jobCh         chan string
}

// NewNotifyWorker creates a NotifyWorker. The job channel is buffered at 64.
func NewNotifyWorker(
	app core.App,
	cfg *config.Config,
	quiClient *quiclient.Client,
	webhookClient *webhookclient.Client,
) *NotifyWorker {
	return &NotifyWorker{
		app:           app,
		cfg:           cfg,
		quiClient:     quiClient,
		webhookClient: webhookClient,
		jobCh:         make(chan string, 64),
	}
}

// Register attaches PocketBase hooks so that any jobs record created or updated
// with status="pending_notify" is dispatched to the worker.
func (w *NotifyWorker) Register() {
	dispatch := func(e *core.RecordEvent) error {
		record := e.Record
		if record.GetString("status") == models.JobStatusPendingNotify {
			select {
			case w.jobCh <- record.Id:
			default:
				w.app.Logger().Warn("notify_worker: job queue full, dropping job", "jobId", record.Id)
			}
		}
		return e.Next()
	}

	w.app.OnRecordAfterCreateSuccess("jobs").BindFunc(dispatch)
	w.app.OnRecordAfterUpdateSuccess("jobs").BindFunc(dispatch)
}

// Start spawns a single goroutine that consumes from the job channel until ctx
// is cancelled.
func (w *NotifyWorker) Start(ctx context.Context) {
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
	w.app.Logger().Info("notify_worker: started")
}

// processJob fetches the record, verifies it is still eligible, sends the
// Discord webhook embed, and marks the job as done.
func (w *NotifyWorker) processJob(_ context.Context, recordID string) {
	record, err := w.app.FindRecordById("jobs", recordID)
	if err != nil {
		w.app.Logger().Error("notify_worker: fetch record failed", "jobId", recordID, "error", err)
		return
	}

	if record.GetString("status") != models.JobStatusPendingNotify {
		return
	}

	record.Set("status", models.JobStatusProcessingNotify)
	if err := w.app.Save(record); err != nil {
		w.app.Logger().Error("notify_worker: claim job failed", "jobId", recordID, "error", err)
		return
	}

	torrentHash := record.GetString("torrent_hash")
	category := record.GetString("category")

	torrent, err := w.quiClient.GetTorrent(torrentHash)
	if err != nil {
		w.failJob(record, "get torrent info: "+err.Error())
		return
	}

	torrentName := torrentHash
	if torrent != nil {
		torrentName = torrent.Name
	} else {
		w.app.Logger().Warn("notify_worker: torrent not found in qui, using hash as name", "jobId", recordID, "torrentHash", torrentHash)
	}

	w.app.Logger().Debug("notify_worker: sending webhook", "jobId", recordID, "torrentName", torrentName, "category", category)

	if err := w.webhookClient.Send(torrentName, category); err != nil {
		w.failJob(record, "send webhook: "+err.Error())
		return
	}

	record.Set("status", models.JobStatusDone)
	record.Set("completed", time.Now())
	if err := w.app.Save(record); err != nil {
		w.app.Logger().Error("notify_worker: save done status failed", "jobId", recordID, "error", err)
		return
	}

	w.app.Logger().Info("notify_worker: job completed", "jobId", recordID, "torrentName", torrentName)
}

// failJob transitions a job record to the error state with a message.
func (w *NotifyWorker) failJob(record *core.Record, msg string) {
	w.app.Logger().Error("notify_worker: job failed", "jobId", record.Id, "error", msg)
	record.Set("status", models.JobStatusError)
	record.Set("error", msg)
	if err := w.app.Save(record); err != nil {
		w.app.Logger().Error("notify_worker: save error state failed", "jobId", record.Id, "error", err)
	}
}
