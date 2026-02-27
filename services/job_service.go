package services

import (
	"github.com/ReinforceZwei/qb-auto/models"
	"github.com/pocketbase/pocketbase/core"
)

// CreateJob inserts a new job record for the given torrent hash and category.
// Anime category jobs start as pending (picked up by title_worker).
// All other jobs (including uncategorized) start as pending_notify (picked up by notify_worker).
func CreateJob(app core.App, torrentHash string, category string) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId("jobs")
	if err != nil {
		return nil, err
	}

	status := models.JobStatusPendingNotify
	if category == "anime" {
		status = models.JobStatusPending
	}

	record := core.NewRecord(collection)
	record.Set("status", status)
	record.Set("torrent_hash", torrentHash)
	record.Set("category", category)

	if err := app.Save(record); err != nil {
		return nil, err
	}

	return record, nil
}
