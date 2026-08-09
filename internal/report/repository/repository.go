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

// ListForAdmin возвращает очередь жалоб вместе с данными, нужными модератору.
// В repository остаётся только преобразование SQL-строк; правила фильтрации и
// границы пагинации проверяет service.
func (r *Repository) ListForAdmin(
	ctx context.Context,
	filter reportmodel.AdminFilter,
) ([]reportmodel.AdminReport, error) {
	rows, err := r.queries.ListReportsForAdmin(ctx, db.ListReportsForAdminParams{
		Status:     filter.Status,
		Reason:     filter.Reason,
		AssigneeID: optionalUUID(filter.AssigneeID),
		PageLimit:  filter.Limit,
		PageOffset: filter.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list reports for admin: %w", err)
	}

	reports := make([]reportmodel.AdminReport, len(rows))
	for index, row := range rows {
		reports[index] = adminReportFromListRow(row)
	}

	return reports, nil
}

func (r *Repository) CountForAdmin(
	ctx context.Context,
	filter reportmodel.AdminFilter,
) (int64, error) {
	total, err := r.queries.CountReportsForAdmin(ctx, db.CountReportsForAdminParams{
		Status:     filter.Status,
		Reason:     filter.Reason,
		AssigneeID: optionalUUID(filter.AssigneeID),
	})
	if err != nil {
		return 0, fmt.Errorf("count reports for admin: %w", err)
	}

	return total, nil
}

func (r *Repository) GetForAdmin(
	ctx context.Context,
	reportID uuid.UUID,
) (reportmodel.AdminReport, error) {
	row, err := r.queries.GetReportForAdmin(ctx, pgUUID(reportID))
	if err != nil {
		return reportmodel.AdminReport{}, fmt.Errorf(
			"get report for admin: %w",
			translateError(err),
		)
	}

	return adminReportFromGetRow(row), nil
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}

func optionalUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}

	return pgUUID(*id)
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

func adminReportFromListRow(row db.ListReportsForAdminRow) reportmodel.AdminReport {
	return adminReportFromFields(
		row.ReportID,
		row.ReportReason,
		row.ReportComment,
		row.ReportStatus,
		row.ReportCreatedAt,
		row.ReporterID,
		row.ReporterNickname,
		row.ReporterPhotoUrl,
		row.OffenderID,
		row.OffenderNickname,
		row.OffenderPhotoUrl,
		row.MessageID,
		row.MessageBody,
		row.MessageCreatedAt,
		row.ExchangeID,
		row.ExchangeStatus,
		row.AssigneeID,
		row.AssigneeNickname,
		row.AssigneePhotoUrl,
	)
}

func adminReportFromGetRow(row db.GetReportForAdminRow) reportmodel.AdminReport {
	return adminReportFromFields(
		row.ReportID,
		row.ReportReason,
		row.ReportComment,
		row.ReportStatus,
		row.ReportCreatedAt,
		row.ReporterID,
		row.ReporterNickname,
		row.ReporterPhotoUrl,
		row.OffenderID,
		row.OffenderNickname,
		row.OffenderPhotoUrl,
		row.MessageID,
		row.MessageBody,
		row.MessageCreatedAt,
		row.ExchangeID,
		row.ExchangeStatus,
		row.AssigneeID,
		row.AssigneeNickname,
		row.AssigneePhotoUrl,
	)
}

func adminReportFromFields(
	reportID pgtype.UUID,
	reason db.ReportReason,
	comment string,
	status db.ReportStatus,
	createdAt pgtype.Timestamptz,
	reporterID pgtype.UUID,
	reporterNickname string,
	reporterPhoto pgtype.Text,
	offenderID pgtype.UUID,
	offenderNickname string,
	offenderPhoto pgtype.Text,
	messageID pgtype.UUID,
	messageBody pgtype.Text,
	messageCreatedAt pgtype.Timestamptz,
	exchangeID pgtype.UUID,
	exchangeStatus db.ChainStatus,
	assigneeID pgtype.UUID,
	assigneeNickname pgtype.Text,
	assigneePhoto pgtype.Text,
) reportmodel.AdminReport {
	report := reportmodel.AdminReport{
		ID:      uuid.UUID(reportID.Bytes),
		Reason:  string(reason),
		Comment: comment,
		Status:  string(status),
		Reporter: reportmodel.AdminUser{
			ID:       uuid.UUID(reporterID.Bytes),
			Nickname: reporterNickname,
			PhotoURL: textPointer(reporterPhoto),
		},
		Offender: reportmodel.AdminUser{
			ID:       uuid.UUID(offenderID.Bytes),
			Nickname: offenderNickname,
			PhotoURL: textPointer(offenderPhoto),
		},
		Message: reportmodel.ReportedMessage{
			ID:        uuid.UUID(messageID.Bytes),
			Body:      messageBody.String,
			CreatedAt: messageCreatedAt.Time,
		},
		Exchange: reportmodel.ReportExchange{
			ID:     uuid.UUID(exchangeID.Bytes),
			Status: string(exchangeStatus),
		},
		CreatedAt: createdAt.Time,
	}

	if assigneeID.Valid {
		report.Assignee = &reportmodel.AdminUser{
			ID:       uuid.UUID(assigneeID.Bytes),
			Nickname: assigneeNickname.String,
			PhotoURL: textPointer(assigneePhoto),
		}
	}

	return report
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
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
