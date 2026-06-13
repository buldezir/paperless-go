package ngxapi

import (
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

func handleListTags(e *core.RequestEvent) error {
	return listNamedRecords(e, "tags", mapTag)
}

func handleGetTag(e *core.RequestEvent) error {
	return getNamedRecord(e, "tags", mapTag)
}

func handleCreateTag(e *core.RequestEvent) error {
	return createNamedRecord(e, "tags", mapTag)
}

func handlePatchTag(e *core.RequestEvent) error {
	return patchNamedRecord(e, "tags", mapTag)
}

func handleDeleteTag(e *core.RequestEvent) error {
	return deleteNamedRecord(e, "tags")
}

func handleListCorrespondents(e *core.RequestEvent) error {
	return listNamedRecords(e, "correspondents", mapCorrespondent)
}

func handleGetCorrespondent(e *core.RequestEvent) error {
	return getNamedRecord(e, "correspondents", mapCorrespondent)
}

func handleCreateCorrespondent(e *core.RequestEvent) error {
	return createNamedRecord(e, "correspondents", mapCorrespondent)
}

func handlePatchCorrespondent(e *core.RequestEvent) error {
	return patchNamedRecord(e, "correspondents", mapCorrespondent)
}

func handleDeleteCorrespondent(e *core.RequestEvent) error {
	return deleteNamedRecord(e, "correspondents")
}

func handleListDocumentTypes(e *core.RequestEvent) error {
	return listNamedRecords(e, "document_types", mapDocumentType)
}

func handleGetDocumentType(e *core.RequestEvent) error {
	return getNamedRecord(e, "document_types", mapDocumentType)
}

func handleCreateDocumentType(e *core.RequestEvent) error {
	return createNamedRecord(e, "document_types", mapDocumentType)
}

func handlePatchDocumentType(e *core.RequestEvent) error {
	return patchNamedRecord(e, "document_types", mapDocumentType)
}

func handleDeleteDocumentType(e *core.RequestEvent) error {
	return deleteNamedRecord(e, "document_types")
}

type recordMapper func(*core.Record) map[string]any

func listNamedRecords(e *core.RequestEvent, collection string, mapper recordMapper) error {
	page, pageSize := paginationParams(e)

	total, err := e.App.CountRecords(collection)
	if err != nil {
		return internalError(e, err)
	}

	offset := (page - 1) * pageSize
	records, err := e.App.FindRecordsByFilter(
		collection,
		"",
		"name",
		pageSize,
		offset,
	)
	if err != nil {
		return internalError(e, err)
	}

	results := make([]any, 0, len(records))
	for _, record := range records {
		results = append(results, mapper(record))
	}

	return paginatedList(e, total, page, pageSize, results)
}

func getNamedRecord(e *core.RequestEvent, collection string, mapper recordMapper) error {
	ngxID, err := parseNgxID(e.Request.PathValue("id"))
	if err != nil {
		return notFound(e, "Not found.")
	}
	record, err := findRecordByNgxID(e.App, collection, ngxID, "", nil)
	if err != nil {
		return notFound(e, "Not found.")
	}
	return writeJSON(e, http.StatusOK, mapper(record))
}

func createNamedRecord(e *core.RequestEvent, collection string, mapper recordMapper) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := e.BindBody(&body); err != nil {
		return badRequest(e, "Invalid request body.")
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		return badRequest(e, "Name is required.")
	}

	coll, err := e.App.FindCollectionByNameOrId(collection)
	if err != nil {
		return internalError(e, err)
	}

	record := core.NewRecord(coll)
	record.Set("name", name)
	if err := e.App.Save(record); err != nil {
		return badRequest(e, err.Error())
	}

	return writeJSON(e, http.StatusCreated, mapper(record))
}

func patchNamedRecord(e *core.RequestEvent, collection string, mapper recordMapper) error {
	ngxID, err := parseNgxID(e.Request.PathValue("id"))
	if err != nil {
		return notFound(e, "Not found.")
	}
	record, err := findRecordByNgxID(e.App, collection, ngxID, "", nil)
	if err != nil {
		return notFound(e, "Not found.")
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := e.BindBody(&body); err != nil {
		return badRequest(e, "Invalid request body.")
	}
	if name := strings.TrimSpace(body.Name); name != "" {
		record.Set("name", name)
	}
	if err := e.App.Save(record); err != nil {
		return badRequest(e, err.Error())
	}

	return writeJSON(e, http.StatusOK, mapper(record))
}

func deleteNamedRecord(e *core.RequestEvent, collection string) error {
	ngxID, err := parseNgxID(e.Request.PathValue("id"))
	if err != nil {
		return notFound(e, "Not found.")
	}
	record, err := findRecordByNgxID(e.App, collection, ngxID, "", nil)
	if err != nil {
		return notFound(e, "Not found.")
	}
	if err := e.App.Delete(record); err != nil {
		return internalError(e, err)
	}
	e.Response.WriteHeader(http.StatusNoContent)
	return nil
}

func handleListTasks(e *core.RequestEvent) error {
	filter := "document.user = {:userId}"
	params := ownerParams(e.Auth.Id)

	if taskID := strings.TrimSpace(e.Request.URL.Query().Get("task_id")); taskID != "" {
		filter += " && task_id = {:taskId}"
		params["taskId"] = taskID
	}

	records, err := e.App.FindRecordsByFilter(
		"processing_jobs",
		filter,
		"-created",
		100,
		0,
		params,
	)
	if err != nil {
		return internalError(e, err)
	}

	results := make([]any, 0, len(records))
	for _, job := range records {
		results = append(results, mapTask(e.App, job))
	}

	return writeJSON(e, http.StatusOK, results)
}
