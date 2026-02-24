package services

import (
	"github.com/ReinforceZwei/qb-auto/models"
	"github.com/pocketbase/pocketbase/core"
)

// CreateJob inserts a new job record with status=pending for the given torrent hash.
// Returns the created record or an error.
func CreateJob(app core.App, torrentHash string) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId("jobs")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)
	record.Set("status", models.JobStatusPending)
	record.Set("torrent_hash", torrentHash)

	if err := app.Save(record); err != nil {
		return nil, err
	}

	return record, nil
}
