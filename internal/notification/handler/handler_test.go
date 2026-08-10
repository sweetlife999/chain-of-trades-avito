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

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	notificationdto "github.com/sweetlife999/chain-of-trades-avito/internal/notification/dto"
	notificationmodel "github.com/sweetlife999/chain-of-trades-avito/internal/notification/model"
	notificationservice "github.com/sweetlife999/chain-of-trades-avito/internal/notification/service"
)

type fakeService struct {
	page         notificationmodel.Page
	listErr      error
	markReadErr  error
	markAllCount int64
	markAllErr   error
	lastUserID   uuid.UUID
	lastFilter   notificationmodel.Filter
	lastMarkedID uuid.UUID
}

func (f *fakeService) List(
	_ context.Context,
	userID uuid.UUID,
	filter notificationmodel.Filter,
) (notificationmodel.Page, error) {
	f.lastUserID = userID
	f.lastFilter = filter
	return f.page, f.listErr
}

func (f *fakeService) MarkRead(
	_ context.Context,
	userID uuid.UUID,
	notificationID uuid.UUID,
) error {
	f.lastUserID = userID
	f.lastMarkedID = notificationID
	return f.markReadErr
}

func (f *fakeService) MarkAllRead(_ context.Context, userID uuid.UUID) (int64, error) {
	f.lastUserID = userID
	return f.markAllCount, f.markAllErr
}

func notificationRouter(service Service, userID *uuid.UUID) http.Handler {
	router := chi.NewRouter()
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if userID != nil {
				r = r.WithContext(authcontext.WithUserID(r.Context(), *userID))
			}
			next.ServeHTTP(w, r)
		})
	}
	New(service).RegisterRoutes(router, auth)
	return router
}

func TestListNotifications(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	notificationID := uuid.New()
	exchangeID := uuid.New()
	createdAt := time.Now().UTC()
	service := &fakeService{page: notificationmodel.Page{
		Notifications: []notificationmodel.Notification{{
			ID: notificationID, ExchangeID: exchangeID, Kind: "exchange_proposed",
			GivesItemTitle: "Кофе", ReceivesItemTitle: "Чай", CreatedAt: createdAt,
		}},
		UnreadCount: 1,
		Limit:       10,
		Offset:      20,
	}}

	request := httptest.NewRequest(
		http.MethodGet,
		"/notifications?unread_only=true&limit=10&offset=20",
		nil,
	)
	response := httptest.NewRecorder()
	notificationRouter(service, &userID).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if service.lastUserID != userID || service.lastFilter != (notificationmodel.Filter{
		UnreadOnly: true, Limit: 10, Offset: 20,
	}) {
		t.Fatalf("user = %v, filter = %+v", service.lastUserID, service.lastFilter)
	}
	var body notificationdto.PageResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.UnreadCount != 1 || len(body.Notifications) != 1 || body.Notifications[0].ID != notificationID.String() {
		t.Fatalf("body = %+v", body)
	}
}

func TestMarkNotificationRead(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	notificationID := uuid.New()
	service := &fakeService{}
	request := httptest.NewRequest(
		http.MethodPost,
		"/notifications/"+notificationID.String()+"/read",
		nil,
	)
	response := httptest.NewRecorder()
	notificationRouter(service, &userID).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if service.lastUserID != userID || service.lastMarkedID != notificationID {
		t.Fatalf("user = %v, notification = %v", service.lastUserID, service.lastMarkedID)
	}
}

func TestMarkAllNotificationsRead(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	service := &fakeService{markAllCount: 3}
	request := httptest.NewRequest(http.MethodPost, "/notifications/read-all", nil)
	response := httptest.NewRecorder()
	notificationRouter(service, &userID).ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "{\"marked_count\":3}\n" {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
}

func TestNotificationHandlerErrors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	tests := []struct {
		name       string
		path       string
		service    *fakeService
		userID     *uuid.UUID
		wantStatus int
	}{
		{name: "unauthorized", path: "/notifications", service: &fakeService{}, wantStatus: http.StatusUnauthorized},
		{name: "bad query", path: "/notifications?limit=none", service: &fakeService{}, userID: &userID, wantStatus: http.StatusBadRequest},
		{name: "invalid id", path: "/notifications/nope/read", service: &fakeService{}, userID: &userID, wantStatus: http.StatusBadRequest},
		{name: "not found", path: "/notifications/" + uuid.NewString() + "/read", service: &fakeService{markReadErr: notificationservice.ErrNotFound}, userID: &userID, wantStatus: http.StatusNotFound},
		{name: "internal", path: "/notifications", service: &fakeService{listErr: errors.New("database unavailable")}, userID: &userID, wantStatus: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			method := http.MethodGet
			if test.name == "invalid id" || test.name == "not found" {
				method = http.MethodPost
			}
			request := httptest.NewRequest(method, test.path, nil)
			response := httptest.NewRecorder()
			notificationRouter(test.service, test.userID).ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}
