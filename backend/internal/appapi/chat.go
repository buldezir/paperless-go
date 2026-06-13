package appapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"paperless-go/backend/internal/ai"
)

type chatRequest struct {
	Messages []ai.ChatMessage `json:"messages"`
}

type chatResponse struct {
	Message ai.ChatMessage `json:"message"`
}

func handleDocumentChat(app core.App, chatter ai.Chatter) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		documentID := strings.TrimSpace(e.Request.PathValue("documentId"))
		if documentID == "" {
			return writeError(e, http.StatusBadRequest, "Document id is required.")
		}

		document, err := app.FindRecordById("documents", documentID)
		if err != nil {
			return writeError(e, http.StatusNotFound, "Document not found.")
		}
		if document.GetString("user") != e.Auth.Id {
			return writeError(e, http.StatusForbidden, "You do not have access to this document.")
		}

		ocrText := strings.TrimSpace(document.GetString("ocr_text"))
		if ocrText == "" {
			return writeError(e, http.StatusBadRequest, "Document has no OCR text yet.")
		}

		var req chatRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return writeError(e, http.StatusBadRequest, "Invalid request body.")
		}
		if len(req.Messages) == 0 {
			return writeError(e, http.StatusBadRequest, "At least one message is required.")
		}

		reply, err := chatter.Chat(context.Background(), ocrText, req.Messages)
		if err != nil {
			return writeError(e, http.StatusInternalServerError, err.Error())
		}

		return writeJSON(e, http.StatusOK, chatResponse{
			Message: ai.ChatMessage{
				Role:    "assistant",
				Content: reply,
			},
		})
	}
}
