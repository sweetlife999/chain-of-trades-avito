package service

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	reportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/report/model"
)

const (
	DefaultAdminLimit int32 = 20
	MaxAdminLimit     int32 = 100
)

var reportStatuses = []string{"open", "resolved", "rejected"}

type AdminRepository interface {
	ListForAdmin(context.Context, reportmodel.AdminFilter) ([]reportmodel.AdminReport, error)
	CountForAdmin(context.Context, reportmodel.AdminFilter) (int64, error)
	GetForAdmin(context.Context, uuid.UUID) (reportmodel.AdminReport, error)
	AssignForAdmin(context.Context, uuid.UUID, uuid.UUID) error
	DecideForAdmin(context.Context, uuid.UUID, uuid.UUID, string, string) error
	RecordMessagesViewed(context.Context, uuid.UUID, uuid.UUID) error
}

// MessageRepository нужен только для чтения треда. AdminService намеренно не получает
// методы создания сообщений или изменения обмена: просмотр жалобы остаётся read-only.
type MessageRepository interface {
	ListMessages(context.Context, uuid.UUID) ([]exchangemodel.Message, error)
}

type AdminService struct {
	reports  AdminRepository
	messages MessageRepository
}

func NewAdmin(reports AdminRepository, messages MessageRepository) *AdminService {
	return &AdminService{reports: reports, messages: messages}
}

func (s *AdminService) List(
	ctx context.Context,
	filter reportmodel.AdminFilter,
) (reportmodel.AdminPage, error) {
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Reason = strings.TrimSpace(filter.Reason)

	if filter.Limit == 0 {
		filter.Limit = DefaultAdminLimit
	}
	if filter.Limit < 1 || filter.Limit > MaxAdminLimit {
		return reportmodel.AdminPage{}, validationError(
			fmt.Sprintf("limit must be between 1 and %d", MaxAdminLimit),
		)
	}
	if filter.Offset < 0 {
		return reportmodel.AdminPage{}, validationError("offset must be non-negative")
	}
	if filter.Status != "" && !slices.Contains(reportStatuses, filter.Status) {
		return reportmodel.AdminPage{}, validationError(
			"status must be one of: " + strings.Join(reportStatuses, ", "),
		)
	}
	if filter.Reason != "" && !slices.Contains(reasons, filter.Reason) {
		return reportmodel.AdminPage{}, validationError(
			"reason must be one of: " + strings.Join(reasons, ", "),
		)
	}

	reports, err := s.reports.ListForAdmin(ctx, filter)
	if err != nil {
		return reportmodel.AdminPage{}, fmt.Errorf("list reports: %w", err)
	}
	total, err := s.reports.CountForAdmin(ctx, filter)
	if err != nil {
		return reportmodel.AdminPage{}, fmt.Errorf("count reports: %w", err)
	}
	if reports == nil {
		reports = []reportmodel.AdminReport{}
	}

	return reportmodel.AdminPage{
		Reports: reports,
		Limit:   filter.Limit,
		Offset:  filter.Offset,
		Total:   total,
	}, nil
}

func (s *AdminService) Get(
	ctx context.Context,
	reportID uuid.UUID,
) (reportmodel.AdminReport, error) {
	if reportID == uuid.Nil {
		return reportmodel.AdminReport{}, validationError("report id is required")
	}

	report, err := s.reports.GetForAdmin(ctx, reportID)
	if err != nil {
		return reportmodel.AdminReport{}, fmt.Errorf("get report: %w", err)
	}

	return report, nil
}

func (s *AdminService) Assign(
	ctx context.Context,
	reportID uuid.UUID,
	adminID uuid.UUID,
) (reportmodel.AdminReport, error) {
	if reportID == uuid.Nil {
		return reportmodel.AdminReport{}, validationError("report id is required")
	}
	if adminID == uuid.Nil {
		return reportmodel.AdminReport{}, validationError("admin id is required")
	}

	if err := s.reports.AssignForAdmin(ctx, reportID, adminID); err != nil {
		return reportmodel.AdminReport{}, fmt.Errorf("assign report: %w", err)
	}

	return s.Get(ctx, reportID)
}

func (s *AdminService) Decide(
	ctx context.Context,
	reportID uuid.UUID,
	adminID uuid.UUID,
	decision string,
	comment string,
) (reportmodel.AdminReport, error) {
	if reportID == uuid.Nil {
		return reportmodel.AdminReport{}, validationError("report id is required")
	}
	if adminID == uuid.Nil {
		return reportmodel.AdminReport{}, validationError("admin id is required")
	}
	if decision != "resolved" && decision != "rejected" {
		return reportmodel.AdminReport{}, validationError(
			"decision must be one of: resolved, rejected",
		)
	}

	comment = strings.TrimSpace(comment)
	if comment == "" {
		return reportmodel.AdminReport{}, validationError("comment is required")
	}
	if utf8.RuneCountInString(comment) > maxCommentLength {
		return reportmodel.AdminReport{}, validationError(
			fmt.Sprintf("comment must be at most %d characters", maxCommentLength),
		)
	}

	if err := s.reports.DecideForAdmin(
		ctx,
		reportID,
		adminID,
		decision,
		comment,
	); err != nil {
		return reportmodel.AdminReport{}, fmt.Errorf("decide report: %w", err)
	}

	return s.Get(ctx, reportID)
}

// ListMessages сначала получает жалобу, а уже из неё — exchange_id. Поэтому админ не
// может передать произвольный exchange_id и читать любой чат через этот endpoint.
func (s *AdminService) ListMessages(
	ctx context.Context,
	reportID uuid.UUID,
	adminID uuid.UUID,
) (reportmodel.AdminReport, []exchangemodel.Message, error) {
	report, err := s.Get(ctx, reportID)
	if err != nil {
		return reportmodel.AdminReport{}, nil, err
	}

	messages, err := s.messages.ListMessages(ctx, report.Exchange.ID)
	if err != nil {
		return reportmodel.AdminReport{}, nil, fmt.Errorf("list report messages: %w", err)
	}
	if messages == nil {
		messages = []exchangemodel.Message{}
	}
	if err := s.reports.RecordMessagesViewed(ctx, reportID, adminID); err != nil {
		return reportmodel.AdminReport{}, nil, fmt.Errorf("record report messages view: %w", err)
	}

	return report, messages, nil
}
