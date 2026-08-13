package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	antiscammodel "github.com/sweetlife999/chain-of-trades-avito/internal/antiscam/model"
)

func TestWorkerPersistsCriticalRuleWhenModelIsDown(t *testing.T) {
	t.Parallel()
	repository := &fakeWorkerRepository{
		job:     antiscammodel.Job{ID: uuid.New(), MessageID: uuid.New(), Attempts: 1},
		message: antiscammodel.Message{ID: uuid.New(), Body: "Пришли код из SMS и CVV"},
	}
	worker := NewWorker(repository, NewAnalyzer(unavailableGenerator{}, "test"))
	processed, err := worker.processNext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !processed || repository.completed == nil || !repository.completed.Suspicious {
		t.Fatalf("completed = %+v", repository.completed)
	}
	if repository.retried {
		t.Fatal("critical rule was retried instead of completed")
	}
}

type unavailableGenerator struct{}

func (unavailableGenerator) Generate(context.Context, string, string, json.RawMessage) (string, error) {
	return "", errors.New("unavailable")
}

type fakeWorkerRepository struct {
	job       antiscammodel.Job
	message   antiscammodel.Message
	completed *antiscammodel.Analysis
	retried   bool
}

func (f *fakeWorkerRepository) Claim(context.Context) (antiscammodel.Job, error) { return f.job, nil }
func (f *fakeWorkerRepository) Input(context.Context, uuid.UUID) (antiscammodel.Message, error) {
	return f.message, nil
}
func (f *fakeWorkerRepository) Context(context.Context, uuid.UUID) ([]antiscammodel.ContextMessage, error) {
	return nil, nil
}
func (f *fakeWorkerRepository) Complete(_ context.Context, _ uuid.UUID, analysis antiscammodel.Analysis) error {
	f.completed = &analysis
	return nil
}
func (f *fakeWorkerRepository) Retry(context.Context, uuid.UUID, int32, error) error {
	f.retried = true
	return nil
}
