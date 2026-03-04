package workers

import (
	"context"
	"time"

	quiclient "github.com/ReinforceZwei/qb-auto/clients/qui"
	webhookclient "github.com/ReinforceZwei/qb-auto/clients/webhook"
	"github.com/ReinforceZwei/qb-auto/config"
	"github.com/ReinforceZwei/qb-auto/models"
	"github.com/pocketbase/pocketbase/core"
)

// NotifyWorker picks up pending_notify jobs and error jobs, sends Discord
// webhook embeds, and transitions pending_notify jobs to done. A single
// goroutine is used; notifications are fast and do not need parallelism.
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
// with status="pending_notify" or status="error" is dispatched to the worker.
func (w *NotifyWorker) Register() {
	dispatch := func(e *core.RecordEvent) error {
		record := e.Record
		status := record.GetString("status")
		if status == models.JobStatusPendingNotify || status == models.JobStatusError {
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

// processJob fetches the record and routes it based on its current status.
func (w *NotifyWorker) processJob(_ context.Context, recordID string) {
	record, err := w.app.FindRecordById("jobs", recordID)
	if err != nil {
		w.app.Logger().Error("notify_worker: fetch record failed", "jobId", recordID, "error", err)
		return
	}

	switch record.GetString("status") {
	case models.JobStatusPendingNotify:
		w.processPendingNotify(record)
	case models.JobStatusError:
		w.processErrorWebhook(record)
	}
}

// processPendingNotify claims the job, sends a success webhook, and marks the
// job done.
func (w *NotifyWorker) processPendingNotify(record *core.Record) {
	record.Set("status", models.JobStatusProcessingNotify)
	if err := w.app.Save(record); err != nil {
		w.app.Logger().Error("notify_worker: claim job failed", "jobId", record.Id, "error", err)
		return
	}

	torrentHash := record.GetString("torrent_hash")
	category := record.GetString("category")
	if category == "" {
		category = "None"
	}

	torrent, err := w.quiClient.GetTorrent(torrentHash)
	if err != nil {
		w.failJob(record, "get torrent info: "+err.Error())
		return
	}

	torrentName := torrentHash
	if torrent != nil {
		torrentName = torrent.Name
	} else {
		w.app.Logger().Warn("notify_worker: torrent not found in qui, using hash as name", "jobId", record.Id, "torrentHash", torrentHash)
	}

	animeTitle := record.GetString("anime_title")

	w.app.Logger().Debug("notify_worker: sending webhook", "jobId", record.Id, "torrentName", torrentName, "category", category, "animeTitle", animeTitle)

	fields := []webhookclient.DiscordField{
		{Name: "Name", Value: torrentName, Inline: false},
		{Name: "Category", Value: category, Inline: true},
	}
	if category == "anime" && animeTitle != "" {
		fields = append(fields, webhookclient.DiscordField{
			Name: "Anime Title", Value: animeTitle, Inline: false,
		})
	}

	embed := webhookclient.DiscordEmbed{
		Title:  "Done: " + torrentName,
		Color:  3447003,
		Fields: fields,
		Footer: webhookclient.DiscordFooter{Text: "qBittorrent"},
	}

	if err := w.webhookClient.Send(embed); err != nil {
		w.failJob(record, "send webhook: "+err.Error())
		return
	}

	record.Set("status", models.JobStatusDone)
	record.Set("completed", time.Now())
	if err := w.app.Save(record); err != nil {
		w.app.Logger().Error("notify_worker: save done status failed", "jobId", record.Id, "error", err)
		return
	}

	w.app.Logger().Info("notify_worker: job completed", "jobId", record.Id, "torrentName", torrentName)
}

// processErrorWebhook sends a failure webhook for a job that has already
// transitioned to the error state (by this or another worker). The record is
// not modified.
func (w *NotifyWorker) processErrorWebhook(record *core.Record) {
	torrentHash := record.GetString("torrent_hash")
	category := record.GetString("category")
	errMsg := record.GetString("error")

	torrentName := torrentHash
	torrent, err := w.quiClient.GetTorrent(torrentHash)
	if err != nil {
		w.app.Logger().Warn("notify_worker: could not fetch torrent for error webhook, using hash as name", "jobId", record.Id, "error", err)
	} else if torrent != nil {
		torrentName = torrent.Name
	}

	w.app.Logger().Debug("notify_worker: sending error webhook", "jobId", record.Id, "torrentName", torrentName, "category", category)

	if category == "" {
		category = "None"
	}
	if torrentName == "" {
		torrentName = "Unknown"
	}

	errEmbed := webhookclient.DiscordEmbed{
		Title: "Failed: " + torrentName,
		Color: 15158332,
		Fields: []webhookclient.DiscordField{
			{Name: "Name", Value: torrentName, Inline: false},
			{Name: "Category", Value: category, Inline: true},
			{Name: "Error", Value: errMsg, Inline: false},
		},
		Footer: webhookclient.DiscordFooter{Text: "qBittorrent"},
	}

	if err := w.webhookClient.Send(errEmbed); err != nil {
		w.app.Logger().Error("notify_worker: send error webhook failed", "jobId", record.Id, "error", err)
	}
}

// failJob transitions a job record to the error state with a message.
// The error webhook is sent by the Register hook reacting to the status change.
func (w *NotifyWorker) failJob(record *core.Record, msg string) {
	w.app.Logger().Error("notify_worker: job failed", "jobId", record.Id, "error", msg)
	record.Set("status", models.JobStatusError)
	record.Set("error", msg)
	if err := w.app.Save(record); err != nil {
		w.app.Logger().Error("notify_worker: save error state failed", "jobId", record.Id, "error", err)
	}
}
