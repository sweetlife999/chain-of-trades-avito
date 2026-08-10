package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	reportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/report/model"
	reportrepository "github.com/sweetlife999/chain-of-trades-avito/internal/report/repository"
)

const (
	maxCommentLength = 2000
	// Жаловаться можно только на сообщение человека: события обмена пишет сам сервис,
	// автора у них нет и отвечать за них некому.
	reportableKind = "text"
)

// Причины повторяют значения report_reason из миграции 00012_reports. Проверяются здесь,
// а не базой: неизвестное значение ENUM Postgres вернул бы как 22P02, то есть 500 вместо 400.
var reasons = []string{"spam", "abuse", "other"}

var (
	ErrValidation       = errors.New("validation error")
	ErrForbidden        = errors.New("report is not allowed")
	ErrNotFound         = reportrepository.ErrNotFound
	ErrDuplicate        = reportrepository.ErrDuplicate
	ErrAlreadyAssigned  = reportrepository.ErrAlreadyAssigned
	ErrNotAssigned      = reportrepository.ErrNotAssigned
	ErrAssignedToOther  = reportrepository.ErrAssignedToOther
	ErrAlreadyProcessed = reportrepository.ErrAlreadyProcessed
)

type Repository interface {
	GetTarget(context.Context, uuid.UUID, uuid.UUID) (reportmodel.Target, error)
	Create(context.Context, reportmodel.NewReport) (reportmodel.Report, error)
}

type Service struct {
	repository Repository
}

type CreateInput struct {
	ReporterID uuid.UUID
	MessageID  uuid.UUID
	Reason     string
	Comment    string
}

type ValidationError struct {
	message string
}

func (e *ValidationError) Error() string {
	return e.message
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}

func New(repository Repository) *Service {
	return &Service{repository: repository}
}

// Create принимает жалобу на сообщение треда. Пожаловаться можно только на чужое текстовое
// сообщение в обмене, где жалобщик участвует, и только один раз — повтор отсекает UNIQUE
// в базе, а не проверка перед вставкой.
func (s *Service) Create(ctx context.Context, input CreateInput) (reportmodel.Report, error) {
	if input.MessageID == uuid.Nil {
		return reportmodel.Report{}, validationError("message_id is required")
	}
	if !slices.Contains(reasons, input.Reason) {
		return reportmodel.Report{}, validationError(
			"reason must be one of: " + strings.Join(reasons, ", "),
		)
	}

	comment := strings.TrimSpace(input.Comment)
	if utf8.RuneCountInString(comment) > maxCommentLength {
		return reportmodel.Report{}, validationError(
			fmt.Sprintf("comment must be at most %d characters", maxCommentLength),
		)
	}

	target, err := s.repository.GetTarget(ctx, input.MessageID, input.ReporterID)
	if err != nil {
		return reportmodel.Report{}, fmt.Errorf("get report target: %w", err)
	}
	if !target.IsParticipant || target.AuthorID == input.ReporterID {
		return reportmodel.Report{}, ErrForbidden
	}
	if target.Kind != reportableKind {
		return reportmodel.Report{}, validationError("only participant messages can be reported")
	}

	report, err := s.repository.Create(ctx, reportmodel.NewReport{
		ReporterID: input.ReporterID,
		MessageID:  input.MessageID,
		Reason:     input.Reason,
		Comment:    comment,
	})
	if err != nil {
		return reportmodel.Report{}, fmt.Errorf("create report: %w", err)
	}

	return report, nil
}

func validationError(message string) error {
	return &ValidationError{message: message}
}
