package config

import (
	"log/slog"
	"sync"

	"github.com/pocketbase/pocketbase/core"
	"paperless-go/backend/internal/ai"
	"paperless-go/backend/internal/ocr"
)

// Snapshot is an immutable view of the live runtime config and clients.
type Snapshot struct {
	Cfg     Config
	OCR     ocr.Provider
	AI      ai.Extractor
	Chatter ai.Chatter
}

// Runtime holds the process-global reloadable config and provider clients.
type Runtime struct {
	mu   sync.RWMutex
	snap Snapshot
}

func NewRuntime() *Runtime {
	return &Runtime{
		snap: Snapshot{Cfg: DefaultsFromEnv()},
	}
}

func (r *Runtime) Snapshot() Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.snap
}

// Reload reads settings from the DB and rebuilds OCR/AI clients.
// If the DB settings are unavailable, falls back to env defaults so the process stays up.
// Missing OCR/AI keys soft-fail: config is still updated and the process stays up.
func (r *Runtime) Reload(app core.App) error {
	cfg, err := Load(app)
	if err != nil {
		app.Logger().Warn("loading app_settings failed; using env defaults", slog.Any("error", err))
		cfg = DefaultsFromEnv()
	}
	r.apply(app, cfg)
	return nil
}

func (r *Runtime) apply(app core.App, cfg Config) {
	logger := app.Logger()
	ocrLogger := logger.With("component", "ocr")
	aiLogger := logger.With("component", "ai")

	var ocrProvider ocr.Provider
	ocrProvider, err := ocr.NewProvider(cfg.OCRProvider, ocr.ProviderConfig{
		GoogleVisionAPIKey: cfg.GoogleVisionAPIKey,
		MistralAPIKey:      cfg.MistralAPIKey,
		MistralModel:       cfg.MistralOCRModel,
		MistralBaseURL:     cfg.MistralAPIBaseURL,
		OCRTimeout:         cfg.OCRTimeout,
		Logger:             ocrLogger,
	})
	if err != nil {
		logger.Warn("OCR provider unavailable after settings reload", slog.Any("error", err))
		ocrProvider = nil
	}

	extractor := ai.NewExtractor(
		cfg.OpenAIAPIKey,
		cfg.OpenAIModel,
		cfg.OpenAIBaseURL,
		cfg.ExtractionPromptVer,
		cfg.ProcessingResultLanguage,
		cfg.OpenAITimeout,
		aiLogger,
	)
	chatter := ai.NewChatter(
		cfg.OpenAIAPIKey,
		cfg.OpenAIChatModel,
		cfg.OpenAIBaseURL,
		cfg.OpenAITimeout,
		aiLogger,
	)

	r.mu.Lock()
	r.snap = Snapshot{
		Cfg:     cfg,
		OCR:     ocrProvider,
		AI:      extractor,
		Chatter: chatter,
	}
	r.mu.Unlock()

	ocrName := "unavailable"
	if ocrProvider != nil {
		ocrName = ocrProvider.Name()
	}
	logger.Info("runtime settings reloaded",
		"ocr", ocrName,
		"ai", extractor.Name(),
		"model", extractor.Model(),
		"chat_model", cfg.OpenAIChatModel,
	)
}

// RegisterHooks seeds defaults, loads runtime state on bootstrap, and reloads on settings changes.
// Bootstrap never fails due to settings — the app must start so admins can open Settings.
func RegisterHooks(app core.App, rt *Runtime) {
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		// App migrations are not applied by serve automatically; apply them here.
		if err := e.App.RunAppMigrations(); err != nil {
			e.App.Logger().Warn("app migrations failed", slog.Any("error", err))
		}

		if err := EnsureDefaults(e.App); err != nil {
			e.App.Logger().Warn("ensure app_settings defaults failed; continuing with env fallback", slog.Any("error", err))
		}

		_ = rt.Reload(e.App)
		return nil
	})

	reload := func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		if e.Record.Id != SingletonID {
			return nil
		}
		_ = rt.Reload(e.App)
		return nil
	}

	app.OnRecordAfterCreateSuccess(CollectionName).BindFunc(reload)
	app.OnRecordAfterUpdateSuccess(CollectionName).BindFunc(reload)
}
