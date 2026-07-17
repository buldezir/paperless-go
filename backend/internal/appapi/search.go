package appapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"paperless-go/backend/internal/ai"
	"paperless-go/backend/internal/config"
)

type searchRequest struct {
	Messages []ai.ChatMessage `json:"messages"`
	Mode     string           `json:"mode"`
}

type searchResponse struct {
	Message   ai.ChatMessage   `json:"message"`
	Documents []ai.DocumentHit `json:"documents"`
}

func handleDeepSearch(app core.App, rt *config.Runtime) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req searchRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return writeError(e, http.StatusBadRequest, "Invalid request body.")
		}
		if len(req.Messages) == 0 {
			return writeError(e, http.StatusBadRequest, "At least one message is required.")
		}

		mode := ai.SearchModeShallow
		if strings.EqualFold(strings.TrimSpace(req.Mode), string(ai.SearchModeDeep)) {
			mode = ai.SearchModeDeep
		}

		agent := rt.Snapshot().SearchAgent
		if agent == nil {
			return writeError(e, http.StatusServiceUnavailable, "AI search is not configured; update Settings.")
		}

		userID := e.Auth.Id
		searcher := func(ctx context.Context, args ai.SearchDocumentsArgs) ([]ai.DocumentHit, error) {
			return searchUserDocuments(app, userID, args)
		}

		reply, hits, err := agent.Search(context.Background(), req.Messages, mode, searcher)
		if err != nil {
			return writeError(e, http.StatusInternalServerError, err.Error())
		}
		if hits == nil {
			hits = []ai.DocumentHit{}
		}

		return writeJSON(e, http.StatusOK, searchResponse{
			Message: ai.ChatMessage{
				Role:    "assistant",
				Content: reply,
			},
			Documents: hits,
		})
	}
}
