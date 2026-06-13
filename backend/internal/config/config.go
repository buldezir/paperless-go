package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	OCRProvider          string
	OCRAPIKey            string
	OCRResultLanguage    string
	OpenCodeGoAPIKey     string
	OpenCodeGoModel      string
	OpenCodeGoBaseURL    string
	OpenCodeGoTimeout    time.Duration
	WorkerPollInterval   time.Duration
	WorkerMaxRetries     int
	ExtractionPromptVer  string
}

func Load() Config {
	timeoutSec, _ := strconv.Atoi(getEnv("OPENCODE_GO_TIMEOUT_SEC", "60"))
	pollSec, _ := strconv.Atoi(getEnv("WORKER_POLL_INTERVAL_SEC", "5"))
	maxRetries, _ := strconv.Atoi(getEnv("WORKER_MAX_RETRIES", "3"))

	return Config{
		OCRProvider:         getEnv("OCR_PROVIDER", "google_vision"),
		OCRAPIKey:           os.Getenv("OCR_API_KEY"),
		OCRResultLanguage:   strings.ToLower(strings.TrimSpace(os.Getenv("OCR_RESULT_LANGUAGE"))),
		OpenCodeGoAPIKey:    os.Getenv("OPENCODE_GO_API_KEY"),
		OpenCodeGoModel:     getEnv("OPENCODE_GO_MODEL", "deepseek-v4-flash"),
		OpenCodeGoBaseURL:   getEnv("OPENCODE_GO_BASE_URL", "https://opencode.ai/zen/go/v1"),
		OpenCodeGoTimeout:   time.Duration(timeoutSec) * time.Second,
		WorkerPollInterval:  time.Duration(pollSec) * time.Second,
		WorkerMaxRetries:    maxRetries,
		ExtractionPromptVer: getEnv("EXTRACTION_PROMPT_VERSION", "v1"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
