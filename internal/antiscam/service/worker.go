package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	antiscammodel "github.com/sweetlife999/chain-of-trades-avito/internal/antiscam/model"
)

const (
	pollInterval    = time.Second
	analysisTimeout = 90 * time.Second
	maxAttempts     = 5
)

type WorkerRepository interface {
	Claim(context.Context) (antiscammodel.Job, error)
	Input(context.Context, uuid.UUID) (antiscammodel.Message, error)
	Context(context.Context, uuid.UUID) ([]antiscammodel.ContextMessage, error)
	Complete(context.Context, uuid.UUID, antiscammodel.Analysis) error
	Retry(context.Context, uuid.UUID, int32, error) error
}

type Worker struct {
	repository WorkerRepository
	analyzer   *Analyzer
}

func NewWorker(repository WorkerRepository, analyzer *Analyzer) *Worker {
	return &Worker{repository: repository, analyzer: analyzer}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		processed, err := w.processNext(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("antiscam worker: %v", err)
		}
		if processed {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) processNext(ctx context.Context) (bool, error) {
	job, err := w.repository.Claim(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	message, err := w.repository.Input(ctx, job.MessageID)
	if err != nil {
		return true, w.retry(ctx, job, err)
	}
	history, err := w.repository.Context(ctx, job.MessageID)
	if err != nil {
		return true, w.retry(ctx, job, err)
	}
	analysisCtx, cancel := context.WithTimeout(ctx, analysisTimeout)
	analysis, err := w.analyzer.Analyze(analysisCtx, message, history)
	cancel()
	if err != nil {
		// Недоступная модель не должна задерживать очевидный критический сигнал.
		// Правила детерминированы, поэтому такой результат можно сохранить сразу;
		// неоднозначные сообщения остаются в очереди и дождутся модели.
		fallback := ruleAnalysis(message.Body)
		fallback.PromptVersion = promptVersion
		finalize(&fallback, nil)
		if fallback.Suspicious {
			if completeErr := w.repository.Complete(ctx, job.ID, fallback); completeErr != nil {
				return true, completeErr
			}
			return true, nil
		}
		return true, w.retry(ctx, job, err)
	}
	if err := w.repository.Complete(ctx, job.ID, analysis); err != nil {
		return true, err
	}
	return true, nil
}

func (w *Worker) retry(ctx context.Context, job antiscammodel.Job, cause error) error {
	if job.Attempts >= maxAttempts {
		// После нескольких сбоев модели не теряем сильные детерминированные сигналы.
		message, inputErr := w.repository.Input(ctx, job.MessageID)
		if inputErr == nil {
			fallback := ruleAnalysis(message.Body)
			fallback.PromptVersion = promptVersion
			finalize(&fallback, nil)
			if completeErr := w.repository.Complete(ctx, job.ID, fallback); completeErr == nil {
				return nil
			}
		}
	}
	delay := int32(1 << min(job.Attempts, 8))
	if err := w.repository.Retry(ctx, job.ID, delay, cause); err != nil {
		return err
	}
	return cause
}
