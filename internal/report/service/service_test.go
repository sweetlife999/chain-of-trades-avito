package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	reportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/report/model"
)

type fakeRepository struct {
	target reportmodel.Target
	err    error
	create func(context.Context, reportmodel.NewReport) (reportmodel.Report, error)
}

func (f *fakeRepository) GetTarget(context.Context, uuid.UUID, uuid.UUID) (reportmodel.Target, error) {
	return f.target, f.err
}

func (f *fakeRepository) Create(
	ctx context.Context,
	report reportmodel.NewReport,
) (reportmodel.Report, error) {
	if f.create == nil {
		return reportmodel.Report{ID: uuid.New(), MessageID: report.MessageID}, nil
	}

	return f.create(ctx, report)
}

// Жалоба принимается ровно в одном сочетании: чужое текстовое сообщение в своём обмене.
// Остальные три строки — запреты, каждый со своим кодом ответа.
func TestCreateGuards(t *testing.T) {
	t.Parallel()

	reporter := uuid.New()
	author := uuid.New()

	tests := []struct {
		name    string
		target  reportmodel.Target
		wantErr error
	}{
		{
			name:   "участник жалуется на чужое сообщение",
			target: reportmodel.Target{Kind: "text", AuthorID: author, IsParticipant: true},
		},
		{
			name:    "посторонний",
			target:  reportmodel.Target{Kind: "text", AuthorID: author, IsParticipant: false},
			wantErr: ErrForbidden,
		},
		{
			name:    "жалоба на собственное сообщение",
			target:  reportmodel.Target{Kind: "text", AuthorID: reporter, IsParticipant: true},
			wantErr: ErrForbidden,
		},
		{
			name:    "жалоба на событие обмена",
			target:  reportmodel.Target{Kind: "exchange_confirmed", IsParticipant: true},
			wantErr: ErrValidation,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := New(&fakeRepository{target: test.target})
			_, err := service.Create(context.Background(), CreateInput{
				ReporterID: reporter,
				MessageID:  uuid.New(),
				Reason:     "abuse",
			})

			if test.wantErr == nil {
				if err != nil {
					t.Fatalf("Create() = %v, want no error", err)
				}
				return
			}
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Create() = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestCreateRejectsBadInput(t *testing.T) {
	t.Parallel()

	reporter := uuid.New()
	valid := reportmodel.Target{Kind: "text", AuthorID: uuid.New(), IsParticipant: true}

	tests := []struct {
		name  string
		input CreateInput
	}{
		{
			name:  "неизвестная причина",
			input: CreateInput{ReporterID: reporter, MessageID: uuid.New(), Reason: "libel"},
		},
		{
			name:  "причина не передана",
			input: CreateInput{ReporterID: reporter, MessageID: uuid.New()},
		},
		{
			name:  "пустой message_id",
			input: CreateInput{ReporterID: reporter, Reason: "spam"},
		},
		{
			name: "комментарий длиннее 2000 символов",
			input: CreateInput{
				ReporterID: reporter,
				MessageID:  uuid.New(),
				Reason:     "spam",
				Comment:    strings.Repeat("я", maxCommentLength+1),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := New(&fakeRepository{target: valid})
			if _, err := service.Create(context.Background(), test.input); !errors.Is(err, ErrValidation) {
				t.Fatalf("Create() = %v, want %v", err, ErrValidation)
			}
		})
	}
}

// Длина считается в символах, а не в байтах: кириллица не должна упираться в лимит вдвое раньше.
func TestCreateAllowsCommentAtLimit(t *testing.T) {
	t.Parallel()

	var saved reportmodel.NewReport
	repository := &fakeRepository{
		target: reportmodel.Target{Kind: "text", AuthorID: uuid.New(), IsParticipant: true},
		create: func(_ context.Context, report reportmodel.NewReport) (reportmodel.Report, error) {
			saved = report
			return reportmodel.Report{ID: uuid.New()}, nil
		},
	}

	comment := strings.Repeat("я", maxCommentLength)
	_, err := New(repository).Create(context.Background(), CreateInput{
		ReporterID: uuid.New(),
		MessageID:  uuid.New(),
		Reason:     "other",
		Comment:    "  " + comment + "  ",
	})
	if err != nil {
		t.Fatalf("Create() = %v, want no error", err)
	}
	if saved.Comment != comment {
		t.Fatalf("saved comment length = %d, want %d", len([]rune(saved.Comment)), maxCommentLength)
	}
}

// Ошибка чтения цели не должна превращаться в 400: она доезжает до хэндлера как есть.
func TestCreatePropagatesTargetError(t *testing.T) {
	t.Parallel()

	want := errors.New("database unavailable")
	service := New(&fakeRepository{err: want})

	_, err := service.Create(context.Background(), CreateInput{
		ReporterID: uuid.New(),
		MessageID:  uuid.New(),
		Reason:     "spam",
	})
	if !errors.Is(err, want) {
		t.Fatalf("Create() = %v, want %v", err, want)
	}
}
