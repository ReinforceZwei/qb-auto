package routes

import (
	"net/http"

	"github.com/ReinforceZwei/qb-auto/models"
	"github.com/pocketbase/pocketbase/core"
)

// retryJobRequest is the JSON body accepted by POST /api/jobs/{id}/retry.
type retryJobRequest struct {
	// Mode selects the restart strategy: "full" re-runs the normal pipeline,
	// "rsync" skips straight to the rsync stage (requires manual title input).
	Mode        string `json:"mode"`
	AnimeTitle  string `json:"anime_title"`
	AnimeListID string `json:"anime_list_id"`
	TMDbID      int    `json:"tmdb_id"`
	TMDbSeason  int    `json:"tmdb_season"`
}

// RegisterJobRoutes registers job management API routes on the serve event.
func RegisterJobRoutes(se *core.ServeEvent) {
	se.Router.POST("/api/jobs/{id}/retry", func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "missing job id",
			})
		}

		record, err := e.App.FindRecordById("jobs", id)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{
				"error": "job not found",
			})
		}

		if record.GetString("status") != models.JobStatusError {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "job is not in error state",
			})
		}

		var req retryJobRequest
		if err := e.BindBody(&req); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request body: " + err.Error(),
			})
		}

		category := record.GetString("category")

		switch req.Mode {
		case "rsync":
			// Skip directly to rsync: requires a manually provided title.
			if category != "anime" {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "skip-to-rsync is only available for anime jobs",
				})
			}
			if req.AnimeTitle == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "missing required field: anime_title",
				})
			}
			record.Set("anime_title", req.AnimeTitle)
			record.Set("anime_list_id", req.AnimeListID)
			if req.TMDbID > 0 {
				record.Set("tmdb_id", req.TMDbID)
			}
			if req.TMDbSeason > 0 {
				record.Set("tmdb_season", req.TMDbSeason)
			}
			record.Set("status", models.JobStatusPendingRsync)
		default:
			// "full": re-run the normal pipeline from the appropriate stage.
			switch {
			case category == "anime" && record.GetString("anime_title") == "":
				record.Set("status", models.JobStatusPending)
			case category == "anime":
				// Title already resolved → continue from the rsync stage.
				record.Set("status", models.JobStatusPendingRsync)
			default:
				record.Set("status", models.JobStatusPendingNotify)
			}
		}

		record.Set("error", "")
		if err := e.App.Save(record); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to save job: " + err.Error(),
			})
		}

		return e.JSON(http.StatusOK, record)
	})
}
