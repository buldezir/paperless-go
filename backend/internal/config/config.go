package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	OCRProvider         string
	OCRAPIKey           string
	OCRResultLanguage   string
	OpenAIAPIKey        string
	OpenAIModel         string
	OpenAIChatModel     string
	OpenAIBaseURL       string
	OpenAITimeout       time.Duration
	WorkerCronExpr      string
	WorkerMaxRetries    int
	ExtractionPromptVer string
}

func Load() Config {
	timeoutSec, _ := strconv.Atoi(getEnv("OPENAI_TIMEOUT_SEC", "60"))
	maxRetries, _ := strconv.Atoi(getEnv("WORKER_MAX_RETRIES", "3"))

	openAIModel := getEnv("OPENAI_MODEL", "gpt-4o-mini")

	return Config{
		OCRProvider:         getEnv("OCR_PROVIDER", "google_vision"),
		OCRAPIKey:           os.Getenv("OCR_API_KEY"),
		OCRResultLanguage:   strings.ToLower(strings.TrimSpace(os.Getenv("OCR_RESULT_LANGUAGE"))),
		OpenAIAPIKey:        os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:         openAIModel,
		OpenAIChatModel:     getEnv("OPENAI_CHAT_MODEL", openAIModel),
		OpenAIBaseURL:       getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAITimeout:       time.Duration(timeoutSec) * time.Second,
		WorkerCronExpr:      getEnv("WORKER_CRON_EXPR", "* * * * *"),
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
