package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	reportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/report/model"
)

type fakeAdminRepository struct {
	reports   []reportmodel.AdminReport
	report    reportmodel.AdminReport
	total     int64
	listErr   error
	countErr  error
	getErr    error
	gotFilter reportmodel.AdminFilter
	gotID     uuid.UUID
}

func (f *fakeAdminRepository) ListForAdmin(
	_ context.Context,
	filter reportmodel.AdminFilter,
) ([]reportmodel.AdminReport, error) {
	f.gotFilter = filter
	return f.reports, f.listErr
}

func (f *fakeAdminRepository) CountForAdmin(
	_ context.Context,
	filter reportmodel.AdminFilter,
) (int64, error) {
	f.gotFilter = filter
	return f.total, f.countErr
}

func (f *fakeAdminRepository) GetForAdmin(
	_ context.Context,
	reportID uuid.UUID,
) (reportmodel.AdminReport, error) {
	f.gotID = reportID
	return f.report, f.getErr
}

type fakeMessageRepository struct {
	messages      []exchangemodel.Message
	err           error
	gotExchangeID uuid.UUID
}

func (f *fakeMessageRepository) ListMessages(
	_ context.Context,
	exchangeID uuid.UUID,
) ([]exchangemodel.Message, error) {
	f.gotExchangeID = exchangeID
	return f.messages, f.err
}

func TestAdminListUsesFiltersAndPagination(t *testing.T) {
	t.Parallel()

	assigneeID := uuid.New()
	repository := &fakeAdminRepository{
		reports: []reportmodel.AdminReport{{ID: uuid.New()}},
		total:   12,
	}
	page, err := NewAdmin(repository, &fakeMessageRepository{}).List(
		context.Background(),
		reportmodel.AdminFilter{
			Status:     " open ",
			Reason:     " abuse ",
			AssigneeID: &assigneeID,
			Limit:      10,
			Offset:     20,
		},
	)
	if err != nil {
		t.Fatalf("List() = %v, want no error", err)
	}
	if repository.gotFilter.Status != "open" || repository.gotFilter.Reason != "abuse" {
		t.Fatalf("filter = %+v", repository.gotFilter)
	}
	if repository.gotFilter.AssigneeID == nil || *repository.gotFilter.AssigneeID != assigneeID {
		t.Fatalf("assignee filter = %v, want %s", repository.gotFilter.AssigneeID, assigneeID)
	}
	if page.Limit != 10 || page.Offset != 20 || page.Total != 12 || len(page.Reports) != 1 {
		t.Fatalf("page = %+v", page)
	}
}

func TestAdminListDefaultsAndNormalizesEmptyResult(t *testing.T) {
	t.Parallel()

	repository := &fakeAdminRepository{}
	page, err := NewAdmin(repository, &fakeMessageRepository{}).List(
		context.Background(),
		reportmodel.AdminFilter{},
	)
	if err != nil {
		t.Fatalf("List() = %v, want no error", err)
	}
	if page.Limit != DefaultAdminLimit || page.Offset != 0 {
		t.Fatalf("pagination = %d/%d", page.Limit, page.Offset)
	}
	if page.Reports == nil || len(page.Reports) != 0 {
		t.Fatalf("reports = %#v, want empty non-nil slice", page.Reports)
	}
}

func TestAdminListRejectsInvalidFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		filter reportmodel.AdminFilter
	}{
		{name: "limit too low", filter: reportmodel.AdminFilter{Limit: -1}},
		{name: "limit too high", filter: reportmodel.AdminFilter{Limit: MaxAdminLimit + 1}},
		{name: "negative offset", filter: reportmodel.AdminFilter{Limit: 1, Offset: -1}},
		{name: "unknown status", filter: reportmodel.AdminFilter{Limit: 1, Status: "pending"}},
		{name: "unknown reason", filter: reportmodel.AdminFilter{Limit: 1, Reason: "fraud"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewAdmin(&fakeAdminRepository{}, &fakeMessageRepository{}).List(
				context.Background(),
				test.filter,
			)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("List() = %v, want %v", err, ErrValidation)
			}
		})
	}
}

func TestAdminListMessagesUsesExchangeFromReport(t *testing.T) {
	t.Parallel()

	reportID := uuid.New()
	exchangeID := uuid.New()
	repository := &fakeAdminRepository{report: reportmodel.AdminReport{
		ID:       reportID,
		Exchange: reportmodel.ReportExchange{ID: exchangeID},
	}}
	messagesRepository := &fakeMessageRepository{
		messages: []exchangemodel.Message{{ID: uuid.New()}},
	}

	report, messages, err := NewAdmin(repository, messagesRepository).ListMessages(
		context.Background(),
		reportID,
	)
	if err != nil {
		t.Fatalf("ListMessages() = %v, want no error", err)
	}
	if report.ID != reportID || repository.gotID != reportID {
		t.Fatalf("report id = %s, repository got %s", report.ID, repository.gotID)
	}
	if messagesRepository.gotExchangeID != exchangeID || len(messages) != 1 {
		t.Fatalf("exchange id = %s, messages = %d", messagesRepository.gotExchangeID, len(messages))
	}
}

func TestAdminGetRejectsEmptyID(t *testing.T) {
	t.Parallel()

	_, err := NewAdmin(&fakeAdminRepository{}, &fakeMessageRepository{}).Get(
		context.Background(),
		uuid.Nil,
	)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Get() = %v, want %v", err, ErrValidation)
	}
}
