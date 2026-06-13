package ngxapi

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

func handleAppConfig(e *core.RequestEvent) error {
	// paperless-ngx returns a list; swift-paperless decodes [ServerConfiguration].
	return writeJSON(e, http.StatusOK, []map[string]any{
		{"id": 1},
	})
}

func handleRemoteVersion(e *core.RequestEvent) error {
	return writeJSON(e, http.StatusOK, map[string]any{
		"version":          "v" + ngxAppVersion,
		"update_available": false,
	})
}

func handleEmptyList(e *core.RequestEvent) error {
	page, pageSize := paginationParams(e)
	return paginatedList(e, 0, page, pageSize, []any{})
}
