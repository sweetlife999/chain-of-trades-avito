package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	antiscammodel "github.com/sweetlife999/chain-of-trades-avito/internal/antiscam/model"
)

func TestAnalyzerRulesCatchCriticalScamWithoutModel(t *testing.T) {
	t.Parallel()
	message := antiscammodel.Message{ID: uuid.New(), Body: "Срочно пришли код из SMS, чтобы подтвердить обмен"}
	analysis, err := NewAnalyzer(nil, "").Analyze(context.Background(), message, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !analysis.Suspicious || analysis.Risk < 90 {
		t.Fatalf("analysis = %+v", analysis)
	}
	if analysis.Category == nil || *analysis.Category != "credentials" {
		t.Fatalf("category = %v", analysis.Category)
	}
}

func TestAnalyzerKeepsCriticalRuleWhenModelDisagrees(t *testing.T) {
	t.Parallel()
	model := &fakeGenerator{response: `{"suspicious":false,"severity":"low","category":"other","reason":"","evidence":[]}`}
	message := antiscammodel.Message{ID: uuid.New(), Body: "Назови CVV и пароль"}
	analysis, err := NewAnalyzer(model, "test").Analyze(context.Background(), message, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !analysis.Suspicious || analysis.Risk != 95 {
		t.Fatalf("analysis = %+v", analysis)
	}
}

func TestAnalyzerDoesNotFlagSafetyWarningWithoutModel(t *testing.T) {
	t.Parallel()
	message := antiscammodel.Message{ID: uuid.New(), Body: "Никому не сообщай код из SMS и не переводи деньги по ссылке"}
	analysis, err := NewAnalyzer(nil, "").Analyze(context.Background(), message, nil)
	if err != nil {
		t.Fatal(err)
	}
	if analysis.Suspicious {
		t.Fatalf("safety warning was flagged: %+v", analysis)
	}
}

func TestAnalyzerOverridesModelFalsePositiveForSafetyWarning(t *testing.T) {
	t.Parallel()
	model := &fakeGenerator{response: `{"suspicious":true,"severity":"high","category":"credentials","reason":"Упоминается код","evidence":["код из SMS"]}`}
	message := antiscammodel.Message{ID: uuid.New(), Body: "Никому не сообщай код из SMS и не переводи деньги до завершения обмена"}
	analysis, err := NewAnalyzer(model, "test").Analyze(context.Background(), message, nil)
	if err != nil {
		t.Fatal(err)
	}
	if analysis.Suspicious || analysis.Risk >= suspiciousRisk {
		t.Fatalf("safety warning was flagged by model: %+v", analysis)
	}
}

func TestAnalyzerDoesNotHideRequestBehindSafetyWarning(t *testing.T) {
	t.Parallel()
	model := &fakeGenerator{response: `{"suspicious":false,"severity":"low","category":"other","reason":"","evidence":[]}`}
	message := antiscammodel.Message{ID: uuid.New(), Body: "Никому не сообщай код, но мне пришли код из SMS"}
	analysis, err := NewAnalyzer(model, "test").Analyze(context.Background(), message, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !analysis.Suspicious || analysis.Risk < 90 {
		t.Fatalf("hidden request was not flagged: %+v", analysis)
	}
}

func TestAnalyzerUsesModelContextAndStrictSchema(t *testing.T) {
	t.Parallel()
	model := &fakeGenerator{response: `{"suspicious":true,"severity":"medium","category":"external_contact","reason":"Уводит общение в мессенджер","evidence":["напиши туда"]}`}
	targetID := uuid.New()
	history := []antiscammodel.ContextMessage{{ID: uuid.New(), AuthorID: uuid.New(), Nickname: "user", Body: "мой телеграм выше"}, {ID: targetID, AuthorID: uuid.New(), Nickname: "suspect", Body: "напиши туда"}}
	analysis, err := NewAnalyzer(model, "test").Analyze(context.Background(), antiscammodel.Message{ID: targetID, Body: "напиши туда"}, history)
	if err != nil {
		t.Fatal(err)
	}
	if !analysis.Suspicious || analysis.Risk != 65 {
		t.Fatalf("analysis = %+v", analysis)
	}
	if len(model.format) == 0 || !json.Valid(model.format) {
		t.Fatal("model did not receive JSON schema")
	}
	if model.user == "" {
		t.Fatal("model did not receive conversation")
	}
}

func TestAnalyzerReturnsModelErrorForWorkerRetry(t *testing.T) {
	t.Parallel()
	want := errors.New("model down")
	_, err := NewAnalyzer(&fakeGenerator{err: want}, "test").Analyze(context.Background(), antiscammodel.Message{ID: uuid.New(), Body: "обычное сообщение"}, nil)
	if !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
}

type fakeGenerator struct {
	response string
	err      error
	user     string
	format   json.RawMessage
}

func (f *fakeGenerator) Generate(_ context.Context, _, user string, format json.RawMessage) (string, error) {
	f.user = user
	f.format = format
	return f.response, f.err
}
