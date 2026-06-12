package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	JobStatusPending     = "pending"
	JobStatusRunning     = "running"
	JobStatusCompleted   = "completed"
	JobStatusFailed      = "failed"
	JobStatusNeedsReview = "needs_review"

	DocStatusPending     = "pending"
	DocStatusProcessing  = "processing"
	DocStatusCompleted   = "completed"
	DocStatusFailed      = "failed"
	DocStatusNeedsReview = "needs_review"

	MetadataSourceAI   = "ai"
	MetadataSourceUser = "user"
)

type ExtractedMetadata struct {
	Title                 string   `json:"title"`
	Purpose               string   `json:"purpose"`
	DocumentDate          string   `json:"document_date"`
	DocumentType          string   `json:"document_type"`
	Tags                  []string `json:"tags"`
	PeopleOrOrganizations []string `json:"people_or_organizations"`
	Summary               string   `json:"summary"`
	Confidence            float64  `json:"confidence"`
}

func (m *ExtractedMetadata) Validate() error {
	if strings.TrimSpace(m.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if m.Confidence < 0 || m.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	if m.DocumentDate != "" {
		if _, err := time.Parse("2006-01-02", m.DocumentDate); err != nil {
			return fmt.Errorf("document_date must be YYYY-MM-DD: %w", err)
		}
	}
	return nil
}

func ParseExtractedMetadata(raw string) (*ExtractedMetadata, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var metadata ExtractedMetadata
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return nil, fmt.Errorf("invalid extraction JSON: %w", err)
	}
	if err := metadata.Validate(); err != nil {
		return nil, err
	}
	return &metadata, nil
}
