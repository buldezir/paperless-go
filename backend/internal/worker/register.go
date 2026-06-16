package worker

import (
	"log"
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
			log.Printf("[worker] cron error: %v", err)
		}
	})

	log.Printf("[worker] registered cron=%q ocr=%s ai=%s model=%s",
		cfg.WorkerCronExpr, ocrProvider.Name(), aiExtractor.Name(), aiExtractor.Model())
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

		_, err := createProcessingJob(e.App, record.Id)
		return err
	})

	p.app.OnRecordCreate("processing_jobs").BindFunc(func(e *core.RecordEvent) error {
		record := e.Record
		if record.GetString("task_id") == "" {
			record.Set("task_id", uuid.New().String())
		}
		if record.Get("retry_count") == nil {
			record.Set("retry_count", 0)
		}
		if record.GetString("job_type") == "" {
			record.Set("job_type", models.JobTypeFull)
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

func createProcessingJob(app core.App, documentID string) (*core.Record, error) {
	jobsCollection, err := app.FindCollectionByNameOrId("processing_jobs")
	if err != nil {
		return nil, err
	}

	job := core.NewRecord(jobsCollection)
	job.Set("document", documentID)
	job.Set("status", models.JobStatusPending)
	job.Set("job_type", models.JobTypeFull)

	if err := app.Save(job); err != nil {
		return nil, err
	}

	log.Printf("[worker] created job=%s document=%s type=%s task_id=%s",
		job.Id, documentID, models.JobTypeFull, job.GetString("task_id"))
	return job, nil
}

func (p *Processor) dispatch(jobID string) {
	if !p.processing.TryLock() {
		return
	}
	go func() {
		defer p.processing.Unlock()
		if err := p.runJob(jobID); err != nil {
			log.Printf("[worker] error: %v", err)
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
	return p.app.RunInTransaction(func(txApp core.App) error {
		job, err := txApp.FindRecordById("processing_jobs", jobID)
		if err != nil {
			return err
		}
		if job.GetString("status") != models.JobStatusPending {
			return nil
		}

		log.Printf("[worker] picked job=%s document=%s type=%s retry=%d",
			job.Id, job.GetString("document"), job.GetString("job_type"), int(job.GetFloat("retry_count")))
		return handleJob(txApp, p.cfg, job, p.ocr, p.ai)
	})
}
