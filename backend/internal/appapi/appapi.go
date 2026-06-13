package appapi

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"paperless-go/backend/internal/ai"
	"paperless-go/backend/internal/config"
)

func Register(app core.App) {
	cfg := config.Load()
	chatter := ai.NewChatter(
		cfg.OpenCodeGoAPIKey,
		cfg.OpenCodeGoChatModel,
		cfg.OpenCodeGoBaseURL,
		cfg.OpenCodeGoTimeout,
	)

	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Priority: 45,
		Func: func(e *core.ServeEvent) error {
			g := e.Router.Group("/api/app")
			g.POST("/documents/{documentId}/chat", bindAuth(handleDocumentChat(app, chatter)))
			return e.Next()
		},
	})
}
