package services

import (
	"github.com/ReinforceZwei/qb-auto/models"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// CreateJob inserts a new job record for the given torrent hash and category.
// Anime category jobs start as pending (picked up by title_worker).
// All other jobs (including uncategorized) start as pending_notify (picked up by notify_worker).
//
// If a job with the same torrent_hash already exists:
//   - Non-error status: the existing record is returned as-is (no duplicate created).
//   - Error status: the job is retried by resetting its status back to pending/pending_notify.
func CreateJob(app core.App, torrentHash string, category string) (*core.Record, error) {
	pendingStatus := models.JobStatusPendingNotify
	if category == "anime" {
		pendingStatus = models.JobStatusPending
	}

	existing, err := app.FindFirstRecordByFilter("jobs", "torrent_hash = {:hash}", dbx.Params{"hash": torrentHash})
	if err == nil {
		// Record found — skip unless it errored, in which case retry.
		if existing.GetString("status") != models.JobStatusError {
			return existing, nil
		}
		existing.Set("status", pendingStatus)
		if err := app.Save(existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	collection, err := app.FindCollectionByNameOrId("jobs")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)
	record.Set("status", pendingStatus)
	record.Set("torrent_hash", torrentHash)
	record.Set("category", category)

	if err := app.Save(record); err != nil {
		return nil, err
	}

	return record, nil
}
