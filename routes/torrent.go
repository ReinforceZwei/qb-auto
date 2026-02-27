package routes

import (
	"net/http"

	"github.com/ReinforceZwei/qb-auto/config"
	"github.com/ReinforceZwei/qb-auto/services"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterTorrentRoutes registers torrent-related API routes on the serve event.
func RegisterTorrentRoutes(se *core.ServeEvent, cfg *config.Config) {
	se.Router.GET("/api/torrent-complete", func(e *core.RequestEvent) error {
		hash := e.Request.URL.Query().Get("hash")
		if hash == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "missing required query parameter: hash",
			})
		}

		category := e.Request.URL.Query().Get("category")
		job, err := services.CreateJob(e.App, hash, category)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}

		return e.JSON(http.StatusOK, map[string]string{
			"job_id":       job.Id,
			"torrent_hash": hash,
			"status":       job.GetString("status"),
		})
	})
}
