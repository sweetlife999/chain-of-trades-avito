package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	reportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/report/model"
)

var (
	ErrNotFound  = errors.New("message not found")
	ErrDuplicate = errors.New("message is already reported")
)

type Repository struct {
	queries *db.Queries
}

func New(queries *db.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) GetTarget(
	ctx context.Context,
	messageID uuid.UUID,
	userID uuid.UUID,
) (reportmodel.Target, error) {
	row, err := r.queries.GetReportTarget(ctx, db.GetReportTargetParams{
		UserID:    pgUUID(userID),
		MessageID: pgUUID(messageID),
	})
	if err != nil {
		return reportmodel.Target{}, fmt.Errorf("get report target: %w", translateError(err))
	}

	return reportmodel.Target{
		Kind:          string(row.Kind),
		AuthorID:      uuid.UUID(row.AuthorID.Bytes),
		IsParticipant: row.IsParticipant,
	}, nil
}

func (r *Repository) Create(
	ctx context.Context,
	report reportmodel.NewReport,
) (reportmodel.Report, error) {
	created, err := r.queries.CreateReport(ctx, db.CreateReportParams{
		ReporterID: pgUUID(report.ReporterID),
		MessageID:  pgUUID(report.MessageID),
		Reason:     db.ReportReason(report.Reason),
		Comment:    report.Comment,
	})
	if err != nil {
		return reportmodel.Report{}, fmt.Errorf("create report: %w", translateError(err))
	}

	return toModel(created), nil
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}

func toModel(report db.Report) reportmodel.Report {
	return reportmodel.Report{
		ID:        uuid.UUID(report.ID.Bytes),
		MessageID: uuid.UUID(report.MessageID.Bytes),
		Reason:    string(report.Reason),
		Comment:   report.Comment,
		Status:    string(report.Status),
		CreatedAt: report.CreatedAt.Time,
	}
}

func translateError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrDuplicate
		case "23503":
			// Сообщение исчезло между проверкой цели и вставкой. Для клиента это то же
			// самое, что «сообщения нет», поэтому и ошибка та же.
			return ErrNotFound
		}
	}

	return err
}
