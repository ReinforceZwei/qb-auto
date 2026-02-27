package models

const (
	JobStatusPending         = "pending"
	JobStatusProcessingTitle = "processing_title"
	JobStatusPendingRsync    = "pending_rsync"
	JobStatusProcessingRsync = "processing_rsync"
	JobStatusPendingNotify    = "pending_notify"
	JobStatusProcessingNotify = "processing_notify"
	JobStatusDone             = "done"
	JobStatusError           = "error"
)
