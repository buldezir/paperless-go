package worker

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	"paperless-go/backend/internal/ai"
	"paperless-go/backend/internal/config"
	"paperless-go/backend/internal/models"
	"paperless-go/backend/internal/ocr"
)

type Processor struct {
	app        core.App
	cfg        config.Config
	ocr        ocr.Provider
	ai         ai.Extractor
	processing sync.Mutex
}

func Register(app core.App) {
	cfg := config.Load()
	ocrProvider, err := ocr.NewProvider(cfg.OCRProvider, ocr.ProviderConfig{
		GoogleVisionAPIKey: cfg.GoogleVisionAPIKey,
		MistralAPIKey:      cfg.MistralAPIKey,
		MistralModel:       cfg.MistralOCRModel,
		MistralBaseURL:     cfg.MistralAPIBaseURL,
		OCRTimeout:         cfg.OCRTimeout,
	})
	if err != nil {
		log.Fatalf("[worker] OCR provider: %v", err)
	}
	aiExtractor := ai.NewExtractor(
		cfg.OpenAIAPIKey,
		cfg.OpenAIModel,
		cfg.OpenAIBaseURL,
		cfg.ExtractionPromptVer,
		cfg.ProcessingResultLanguage,
		cfg.OpenAITimeout,
	)

	p := &Processor{
		app: app,
		cfg: cfg,
		ocr: ocrProvider,
		ai:  aiExtractor,
	}
	p.registerHooks()

	app.Cron().MustAdd("process_pending_jobs", cfg.WorkerCronExpr, func() {
		if err := p.processNextPending(); err != nil {
			app.Logger().Error("cron error", slog.Any("error", err))
		}
	})

	app.Logger().Info("worker registered",
		"cron", cfg.WorkerCronExpr,
		"ocr", ocrProvider.Name(),
		"ai", aiExtractor.Name(),
		"model", aiExtractor.Model(),
	)
}

func (p *Processor) registerHooks() {
	p.app.OnRecordCreate("documents").BindFunc(func(e *core.RecordEvent) error {
		record := e.Record
		if record.GetString("processing_status") == "" {
			record.Set("processing_status", models.DocStatusPending)
		}
		if err := e.Next(); err != nil {
			return err
		}

		_, err := createProcessingJob(e.App, record.Id, models.FullPipelineSteps, nil)
		return err
	})

	p.app.OnRecordCreate("processing_jobs").BindFunc(func(e *core.RecordEvent) error {
		record := e.Record
		if record.GetString("task_id") == "" {
			record.Set("task_id", uuid.New().String())
		}
		steps, err := parseSteps(record)
		if err != nil {
			return err
		}
		if len(steps) == 0 {
			record.Set("steps", models.FullPipelineSteps)
		}
		return e.Next()
	})

	p.app.OnRecordDelete("documents").BindFunc(func(e *core.RecordEvent) error {
		jobs, err := e.App.FindRecordsByFilter(
			"processing_jobs",
			"document = {:docId}",
			"-created",
			100,
			0,
			map[string]any{"docId": e.Record.Id},
		)
		if err != nil {
			return err
		}

		for _, job := range jobs {
			if err := e.App.Delete(job); err != nil {
				return err
			}
		}

		return e.Next()
	})

	p.app.OnRecordAfterCreateSuccess("processing_jobs").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("status") == models.JobStatusPending {
			go p.dispatch(e.Record.Id)
		}
		return e.Next()
	})

	p.app.OnRecordAfterUpdateSuccess("processing_jobs").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("status") == models.JobStatusPending {
			go p.dispatch(e.Record.Id)
		}
		return e.Next()
	})
}

func createProcessingJob(app core.App, documentID string, steps []string, forceSteps []string) (*core.Record, error) {
	jobsCollection, err := app.FindCollectionByNameOrId("processing_jobs")
	if err != nil {
		return nil, err
	}

	job := core.NewRecord(jobsCollection)
	job.Set("document", documentID)
	job.Set("status", models.JobStatusPending)
	job.Set("steps", steps)
	if len(forceSteps) > 0 {
		job.Set("force_steps", forceSteps)
	}

	if err := app.Save(job); err != nil {
		return nil, err
	}

	app.Logger().Info("created job",
		"job", job.Id,
		"document", documentID,
		"steps", steps,
		"task_id", job.GetString("task_id"),
	)
	return job, nil
}

func (p *Processor) dispatch(jobID string) {
	if !p.processing.TryLock() {
		return
	}
	go func() {
		defer p.processing.Unlock()
		if err := p.runJob(jobID); err != nil {
			p.app.Logger().Error("job error", "job", jobID, slog.Any("error", err))
		}
	}()
}

func (p *Processor) processNextPending() error {
	if !p.processing.TryLock() {
		return nil
	}
	defer p.processing.Unlock()

	jobs, err := p.app.FindRecordsByFilter(
		"processing_jobs",
		"status = {:status}",
		"created",
		1,
		0,
		map[string]any{"status": models.JobStatusPending},
	)
	if err != nil {
		return err
	}
	if len(jobs) == 0 {
		return nil
	}

	return p.runJob(jobs[0].Id)
}

func (p *Processor) runJob(jobID string) error {
	claimed := false
	err := p.app.RunInTransaction(func(txApp core.App) error {
		job, err := txApp.FindRecordById("processing_jobs", jobID)
		if err != nil {
			return err
		}
		if job.GetString("status") != models.JobStatusPending {
			return nil
		}

		steps, err := parseSteps(job)
		if err != nil {
			return err
		}
		if len(steps) == 0 {
			return fmt.Errorf("job %s has no steps", jobID)
		}

		document, err := txApp.FindRecordById("documents", job.GetString("document"))
		if err != nil {
			return err
		}

		claimed = true
		p.app.Logger().Info("picked job",
			"job", job.Id,
			"document", document.Id,
			"steps", steps,
		)

		job.Set("status", models.JobStatusRunning)
		if job.GetString("started_at") == "" {
			job.Set("started_at", nowTimestamp())
		}

		runs, err := parseStepRuns(job)
		if err != nil {
			return err
		}
		runs = syncStepRuns(steps, runs)
		if len(runs) == 0 {
			runs = initStepRuns(steps)
		}
		saveStepRuns(job, runs)

		document.Set("processing_status", models.DocStatusProcessing)
		if err := txApp.Save(document); err != nil {
			return err
		}
		return txApp.Save(job)
	})
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}

	runner := NewPipelineRunner(p.app, p.cfg, p.ocr, p.ai)
	return runner.Run(context.Background(), jobID)
}
