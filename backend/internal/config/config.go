package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	OCRProvider              string
	GoogleVisionAPIKey       string
	MistralAPIKey            string
	MistralOCRModel          string
	MistralAPIBaseURL        string
	ProcessingResultLanguage string
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
		OCRProvider:              getEnv("OCR_PROVIDER", "google_vision"),
		GoogleVisionAPIKey:       os.Getenv("GOOGLE_VISION_API_KEY"),
		MistralAPIKey:            os.Getenv("MISTRAL_API_KEY"),
		MistralOCRModel:          getEnv("MISTRAL_OCR_MODEL", "mistral-ocr-latest"),
		MistralAPIBaseURL:        getEnv("MISTRAL_API_BASE_URL", "https://api.mistral.ai/v1"),
		ProcessingResultLanguage: strings.ToLower(strings.TrimSpace(os.Getenv("PROCESSING_RESULT_LANGUAGE"))),
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
