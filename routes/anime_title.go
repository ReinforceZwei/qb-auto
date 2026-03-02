package routes

import (
	"net/http"

	"github.com/ReinforceZwei/qb-auto/clients/animelist"
	tmdbclient "github.com/ReinforceZwei/qb-auto/clients/tmdb"
	"github.com/ReinforceZwei/qb-auto/llm"
	"github.com/ReinforceZwei/qb-auto/services"
	"github.com/pocketbase/pocketbase/core"
)

// resolveAnimeTitleRequest is the JSON body accepted by POST /api/resolve-anime-title.
type resolveAnimeTitleRequest struct {
	FolderName      string `json:"folder_name"`
	SearchAnimeList bool   `json:"search_anime_list"`
}

// resolveAnimeTitleResponse is the JSON body returned on success.
// AnimeListID is omitted from the response when SearchAnimeList was false.
type resolveAnimeTitleResponse struct {
	AnimeTitle  string `json:"anime_title"`
	TMDbID      int    `json:"tmdb_id"`
	AnimeListID *int   `json:"anime_list_id,omitempty"`
}

// RegisterAnimeTitleRoutes registers the resolve-anime-title route on the serve event.
func RegisterAnimeTitleRoutes(
	se *core.ServeEvent,
	llmClient *llm.Client,
	tmdbClient *tmdbclient.Client,
	animeListClient *animelist.Client,
) {
	se.Router.POST("/api/resolve-anime-title", func(e *core.RequestEvent) error {
		var req resolveAnimeTitleRequest
		if err := e.BindBody(&req); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request body: " + err.Error(),
			})
		}
		if req.FolderName == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "missing required field: folder_name",
			})
		}

		resolved, err := services.ResolveAnimeTitle(e.Request.Context(), req.FolderName, llmClient, tmdbClient)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}

		resp := resolveAnimeTitleResponse{
			AnimeTitle: resolved.AnimeTitle,
			TMDbID:     resolved.TMDbID,
		}

		if req.SearchAnimeList {
			records, err := animeListClient.Search(resolved.AnimeTitle)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "anime list search: " + err.Error(),
				})
			}
			if len(records) > 0 {
				id := records[0].ID
				resp.AnimeListID = &id
			}
		}

		return e.JSON(http.StatusOK, resp)
	})
}
