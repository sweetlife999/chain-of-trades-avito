package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authmiddleware "github.com/sweetlife999/chain-of-trades-avito/internal/auth/middleware"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	reportdto "github.com/sweetlife999/chain-of-trades-avito/internal/report/dto"
	reportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/report/model"
	reportservice "github.com/sweetlife999/chain-of-trades-avito/internal/report/service"
)

type fakeAdminReadService struct {
	page        reportmodel.AdminPage
	report      reportmodel.AdminReport
	messages    []exchangemodel.Message
	err         error
	called      bool
	gotFilter   reportmodel.AdminFilter
	gotReportID uuid.UUID
}

func (f *fakeAdminReadService) List(
	_ context.Context,
	filter reportmodel.AdminFilter,
) (reportmodel.AdminPage, error) {
	f.called = true
	f.gotFilter = filter
	return f.page, f.err
}

func (f *fakeAdminReadService) Get(
	_ context.Context,
	reportID uuid.UUID,
) (reportmodel.AdminReport, error) {
	f.called = true
	f.gotReportID = reportID
	return f.report, f.err
}

func (f *fakeAdminReadService) ListMessages(
	_ context.Context,
	reportID uuid.UUID,
) (reportmodel.AdminReport, []exchangemodel.Message, error) {
	f.called = true
	f.gotReportID = reportID
	return f.report, f.messages, f.err
}

func adminReportRouter(service AdminReadService) http.Handler {
	router := chi.NewRouter()
	NewAdmin(service).RegisterRoutes(router)
	return router
}

func TestAdminReportListParsesFiltersAndReturnsPage(t *testing.T) {
	t.Parallel()

	assigneeID := uuid.New()
	reportID := uuid.New()
	service := &fakeAdminReadService{page: reportmodel.AdminPage{
		Reports: []reportmodel.AdminReport{{
			ID:       reportID,
			Reporter: reportmodel.AdminUser{ID: uuid.New(), Nickname: "reporter"},
			Offender: reportmodel.AdminUser{ID: uuid.New(), Nickname: "offender"},
			Message:  reportmodel.ReportedMessage{ID: uuid.New()},
			Exchange: reportmodel.ReportExchange{ID: uuid.New(), Status: "proposed"},
		}},
		Limit: 10, Offset: 5, Total: 42,
	}}
	request := httptest.NewRequest(
		http.MethodGet,
		"/reports?status=open&reason=abuse&assignee_id="+assigneeID.String()+"&limit=10&offset=5",
		nil,
	)
	response := httptest.NewRecorder()
	adminReportRouter(service).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}
	if service.gotFilter.Status != "open" || service.gotFilter.Reason != "abuse" ||
		service.gotFilter.Limit != 10 || service.gotFilter.Offset != 5 ||
		service.gotFilter.AssigneeID == nil || *service.gotFilter.AssigneeID != assigneeID {
		t.Fatalf("filter = %+v", service.gotFilter)
	}

	var body reportdto.AdminReportListResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Reports) != 1 || body.Reports[0].ID != reportID.String() ||
		body.Pagination.Total != 42 {
		t.Fatalf("body = %+v", body)
	}
}

func TestAdminReportDetailAndMessages(t *testing.T) {
	t.Parallel()

	reportID := uuid.New()
	exchangeID := uuid.New()
	messageID := uuid.New()
	service := &fakeAdminReadService{
		report: reportmodel.AdminReport{
			ID:       reportID,
			Reporter: reportmodel.AdminUser{ID: uuid.New(), Nickname: "reporter"},
			Offender: reportmodel.AdminUser{ID: uuid.New(), Nickname: "offender"},
			Message:  reportmodel.ReportedMessage{ID: messageID, Body: "reported"},
			Exchange: reportmodel.ReportExchange{ID: exchangeID, Status: "confirmed"},
		},
		messages: []exchangemodel.Message{{
			ID: messageID, Kind: "text", CreatedAt: time.Now().UTC(),
		}},
	}

	detail := httptest.NewRecorder()
	adminReportRouter(service).ServeHTTP(
		detail,
		httptest.NewRequest(http.MethodGet, "/reports/"+reportID.String(), nil),
	)
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status = %d; body = %s", detail.Code, detail.Body.String())
	}

	thread := httptest.NewRecorder()
	adminReportRouter(service).ServeHTTP(
		thread,
		httptest.NewRequest(http.MethodGet, "/reports/"+reportID.String()+"/messages", nil),
	)
	if thread.Code != http.StatusOK {
		t.Fatalf("messages status = %d; body = %s", thread.Code, thread.Body.String())
	}
	var body reportdto.AdminReportMessagesResponse
	if err := json.NewDecoder(thread.Body).Decode(&body); err != nil {
		t.Fatalf("decode messages: %v", err)
	}
	if body.ReportID != reportID.String() || body.ExchangeID != exchangeID.String() ||
		len(body.Messages) != 1 {
		t.Fatalf("messages body = %+v", body)
	}
}

func TestAdminReportErrors(t *testing.T) {
	t.Parallel()

	reportID := uuid.NewString()
	tests := []struct {
		name       string
		path       string
		serviceErr error
		wantStatus int
	}{
		{name: "invalid assignee", path: "/reports?assignee_id=bad", wantStatus: http.StatusBadRequest},
		{name: "invalid limit", path: "/reports?limit=bad", wantStatus: http.StatusBadRequest},
		{name: "invalid offset", path: "/reports?offset=bad", wantStatus: http.StatusBadRequest},
		{name: "invalid report id", path: "/reports/bad", wantStatus: http.StatusBadRequest},
		{name: "validation", path: "/reports", serviceErr: reportservice.ErrValidation, wantStatus: http.StatusBadRequest},
		{name: "not found", path: "/reports/" + reportID, serviceErr: reportservice.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "internal", path: "/reports", serviceErr: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			service := &fakeAdminReadService{err: test.serviceErr}
			response := httptest.NewRecorder()
			adminReportRouter(service).ServeHTTP(
				response,
				httptest.NewRequest(http.MethodGet, test.path, nil),
			)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

type reportTokenParser struct{ userID uuid.UUID }

func (f reportTokenParser) Parse(string) (uuid.UUID, error) { return f.userID, nil }

type reportAdminChecker struct{ admin bool }

func (f reportAdminChecker) IsAdmin(context.Context, uuid.UUID) (bool, error) { return f.admin, nil }

func TestAdminReportRoutesRequireAdministrator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cookie     bool
		admin      bool
		wantStatus int
		wantCall   bool
	}{
		{name: "unauthenticated", wantStatus: http.StatusUnauthorized},
		{name: "regular user", cookie: true, wantStatus: http.StatusForbidden},
		{name: "administrator", cookie: true, admin: true, wantStatus: http.StatusOK, wantCall: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			service := &fakeAdminReadService{page: reportmodel.AdminPage{Reports: []reportmodel.AdminReport{}}}
			authenticator := authmiddleware.New(reportTokenParser{userID: uuid.New()})
			authorizer := authmiddleware.NewAdminAuthorizer(reportAdminChecker{admin: test.admin})
			router := chi.NewRouter()
			router.Route("/admin", func(admin chi.Router) {
				admin.Use(authenticator.RequireAuthentication)
				admin.Use(authorizer.RequireAdmin)
				NewAdmin(service).RegisterRoutes(admin)
			})

			request := httptest.NewRequest(http.MethodGet, "/admin/reports", nil)
			if test.cookie {
				request.AddCookie(&http.Cookie{Name: authmiddleware.CookieName, Value: "valid-token"})
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			if service.called != test.wantCall {
				t.Fatalf("service called = %v, want %v", service.called, test.wantCall)
			}
		})
	}
}
