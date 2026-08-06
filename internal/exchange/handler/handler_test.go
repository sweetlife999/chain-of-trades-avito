package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	exchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/service"
)

func TestListReturnsExchanges(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	want := exchangeDetails(userID)
	service := &fakeService{list: func(_ context.Context, actualUserID uuid.UUID) ([]exchangemodel.Details, error) {
		if actualUserID != userID {
			t.Fatalf("ListForUser() user ID = %s, want %s", actualUserID, userID)
		}
		return []exchangemodel.Details{want}, nil
	}}

	response := performRequest(service, http.MethodGet, "/exchanges", authenticateAs(userID))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	body := response.Body.String()
	for _, field := range []string{"participants", "gives_item", "receives_item", "nickname"} {
		if !strings.Contains(body, `"`+field+`"`) {
			t.Fatalf("response does not contain %q: %s", field, body)
		}
	}

	if strings.Contains(body, `"chain`) {
		t.Fatalf("response exposes internal chain terminology: %s", body)
	}
}

func TestListReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	service := &fakeService{list: func(context.Context, uuid.UUID) ([]exchangemodel.Details, error) {
		return nil, nil
	}}

	response := performRequest(service, http.MethodGet, "/exchanges", authenticateAs(uuid.New()))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	if response.Body.String() != "[]\n" {
		t.Fatalf("body = %q, want []", response.Body.String())
	}
}

func TestGetByIDReturnsExchange(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	want := exchangeDetails(userID)
	service := &fakeService{get: func(
		_ context.Context,
		exchangeID uuid.UUID,
		actualUserID uuid.UUID,
	) (exchangemodel.Details, error) {
		if exchangeID != want.ID {
			t.Fatalf("GetForUser() exchange ID = %s, want %s", exchangeID, want.ID)
		}
		if actualUserID != userID {
			t.Fatalf("GetForUser() user ID = %s, want %s", actualUserID, userID)
		}
		return want, nil
	}}

	response := performRequest(
		service,
		http.MethodGet,
		"/exchanges/"+want.ID.String(),
		authenticateAs(userID),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	if !strings.Contains(response.Body.String(), want.ID.String()) {
		t.Fatalf("response does not contain exchange ID: %s", response.Body.String())
	}
}

func TestRoutesRequireAuthentication(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		list: func(context.Context, uuid.UUID) ([]exchangemodel.Details, error) {
			t.Fatal("ListForUser() must not be called")
			return nil, nil
		},
		get: func(context.Context, uuid.UUID, uuid.UUID) (exchangemodel.Details, error) {
			t.Fatal("GetForUser() must not be called")
			return exchangemodel.Details{}, nil
		},
	}

	for _, path := range []string{"/exchanges", "/exchanges/" + uuid.New().String()} {
		response := performRequest(service, http.MethodGet, path, passThroughAuth)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("path %s: status = %d, want %d", path, response.Code, http.StatusUnauthorized)
		}
	}
}

func TestGetByIDErrorStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		serviceErr error
		wantStatus int
	}{
		{name: "invalid UUID", path: "/exchanges/not-a-uuid", wantStatus: http.StatusBadRequest},
		{name: "forbidden", path: "/exchanges/" + uuid.New().String(), serviceErr: exchangeservice.ErrForbidden, wantStatus: http.StatusForbidden},
		{name: "not found", path: "/exchanges/" + uuid.New().String(), serviceErr: exchangeservice.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "internal error", path: "/exchanges/" + uuid.New().String(), serviceErr: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := &fakeService{get: func(
				context.Context,
				uuid.UUID,
				uuid.UUID,
			) (exchangemodel.Details, error) {
				return exchangemodel.Details{}, test.serviceErr
			}}

			response := performRequest(service, http.MethodGet, test.path, authenticateAs(uuid.New()))
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func TestListInternalError(t *testing.T) {
	t.Parallel()

	service := &fakeService{list: func(context.Context, uuid.UUID) ([]exchangemodel.Details, error) {
		return nil, errors.New("database unavailable")
	}}

	response := performRequest(service, http.MethodGet, "/exchanges", authenticateAs(uuid.New()))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
}

func TestParticipationDecisionRoutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		set  func(*fakeService, func(context.Context, uuid.UUID, uuid.UUID) error)
	}{
		{
			name: "confirm",
			path: "/exchanges/%s/confirm",
			set: func(service *fakeService, operation func(context.Context, uuid.UUID, uuid.UUID) error) {
				service.confirm = operation
			},
		},
		{
			name: "decline",
			path: "/exchanges/%s/decline",
			set: func(service *fakeService, operation func(context.Context, uuid.UUID, uuid.UUID) error) {
				service.decline = operation
			},
		},
		{
			name: "complete",
			path: "/exchanges/%s/complete",
			set: func(service *fakeService, operation func(context.Context, uuid.UUID, uuid.UUID) error) {
				service.complete = operation
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			exchangeID := uuid.New()
			userID := uuid.New()
			service := &fakeService{}
			test.set(service, func(
				_ context.Context,
				actualExchangeID uuid.UUID,
				actualUserID uuid.UUID,
			) error {
				if actualExchangeID != exchangeID || actualUserID != userID {
					t.Fatalf(
						"decision args = (%s, %s), want (%s, %s)",
						actualExchangeID,
						actualUserID,
						exchangeID,
						userID,
					)
				}
				return nil
			})

			response := performRequest(
				service,
				http.MethodPost,
				fmt.Sprintf(test.path, exchangeID),
				authenticateAs(userID),
			)
			if response.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusNoContent, response.Body.String())
			}
		})
	}
}

func TestParticipationDecisionErrorStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pathID     string
		serviceErr error
		wantStatus int
	}{
		{name: "invalid UUID", pathID: "not-a-uuid", wantStatus: http.StatusBadRequest},
		{name: "forbidden", pathID: uuid.New().String(), serviceErr: exchangeservice.ErrForbidden, wantStatus: http.StatusForbidden},
		{name: "not found", pathID: uuid.New().String(), serviceErr: exchangeservice.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "conflict", pathID: uuid.New().String(), serviceErr: exchangeservice.ErrConflict, wantStatus: http.StatusConflict},
		{name: "internal", pathID: uuid.New().String(), serviceErr: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := &fakeService{confirm: func(context.Context, uuid.UUID, uuid.UUID) error {
				return test.serviceErr
			}}
			response := performRequest(
				service,
				http.MethodPost,
				"/exchanges/"+test.pathID+"/confirm",
				authenticateAs(uuid.New()),
			)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func TestParticipationDecisionRoutesRequireAuthentication(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		confirm: func(context.Context, uuid.UUID, uuid.UUID) error {
			t.Fatal("ConfirmParticipation() must not be called")
			return nil
		},
		decline: func(context.Context, uuid.UUID, uuid.UUID) error {
			t.Fatal("DeclineParticipation() must not be called")
			return nil
		},
		complete: func(context.Context, uuid.UUID, uuid.UUID) error {
			t.Fatal("CompleteParticipation() must not be called")
			return nil
		},
	}

	for _, action := range []string{"confirm", "decline", "complete"} {
		response := performRequest(
			service,
			http.MethodPost,
			"/exchanges/"+uuid.New().String()+"/"+action,
			passThroughAuth,
		)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("action %s: status = %d, want %d", action, response.Code, http.StatusUnauthorized)
		}
	}
}

func TestPostMessageReturnsCreatedMessage(t *testing.T) {
	t.Parallel()

	exchangeID := uuid.New()
	userID := uuid.New()
	body := "заберу в субботу"
	service := &fakeService{postMessage: func(
		_ context.Context,
		actualExchangeID uuid.UUID,
		actualUserID uuid.UUID,
		actualBody string,
	) (exchangemodel.Message, error) {
		if actualExchangeID != exchangeID || actualUserID != userID || actualBody != body {
			t.Fatalf("PostMessage() args = (%s, %s, %q)", actualExchangeID, actualUserID, actualBody)
		}
		return exchangemodel.Message{
			ID:        uuid.New(),
			Kind:      "text",
			Body:      &body,
			Author:    &exchangemodel.ParticipantUser{ID: userID, Nickname: "samir"},
			CreatedAt: time.Now(),
		}, nil
	}}

	response := performRequestWithBody(
		service,
		http.MethodPost,
		"/exchanges/"+exchangeID.String()+"/messages",
		strings.NewReader(`{"body":"`+body+`"}`),
		authenticateAs(userID),
	)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}

	for _, field := range []string{"kind", "body", "author", "created_at"} {
		if !strings.Contains(response.Body.String(), `"`+field+`"`) {
			t.Fatalf("response does not contain %q: %s", field, response.Body.String())
		}
	}
}

func TestPostMessageRejectsBrokenJSON(t *testing.T) {
	t.Parallel()

	response := performRequestWithBody(
		&fakeService{},
		http.MethodPost,
		"/exchanges/"+uuid.New().String()+"/messages",
		strings.NewReader("{"),
		authenticateAs(uuid.New()),
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestMessageErrorStatuses(t *testing.T) {
	t.Parallel()

	cases := map[error]int{
		exchangeservice.ErrValidation:  http.StatusBadRequest,
		exchangeservice.ErrForbidden:   http.StatusForbidden,
		exchangeservice.ErrNotFound:    http.StatusNotFound,
		exchangeservice.ErrConflict:    http.StatusConflict,
		errors.New("database is down"): http.StatusInternalServerError,
	}

	for serviceError, want := range cases {
		serviceError, want := serviceError, want
		t.Run(fmt.Sprintf("%v", serviceError), func(t *testing.T) {
			t.Parallel()

			service := &fakeService{postMessage: func(
				context.Context,
				uuid.UUID,
				uuid.UUID,
				string,
			) (exchangemodel.Message, error) {
				return exchangemodel.Message{}, serviceError
			}}

			response := performRequestWithBody(
				service,
				http.MethodPost,
				"/exchanges/"+uuid.New().String()+"/messages",
				strings.NewReader(`{"body":"привет"}`),
				authenticateAs(uuid.New()),
			)
			if response.Code != want {
				t.Fatalf("status = %d, want %d", response.Code, want)
			}
		})
	}
}

func TestListMessagesReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	service := &fakeService{listMessages: func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
	) ([]exchangemodel.Message, error) {
		return nil, nil
	}}

	response := performRequest(
		service,
		http.MethodGet,
		"/exchanges/"+uuid.New().String()+"/messages",
		authenticateAs(uuid.New()),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	if response.Body.String() != "[]\n" {
		t.Fatalf("body = %q, want []", response.Body.String())
	}
}

func TestMessageRoutesRequireAuthentication(t *testing.T) {
	t.Parallel()

	path := "/exchanges/" + uuid.New().String() + "/messages"

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		response := performRequestWithBody(
			&fakeService{},
			method,
			path,
			strings.NewReader(`{"body":"привет"}`),
			passThroughAuth,
		)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s: status = %d, want %d", method, response.Code, http.StatusUnauthorized)
		}
	}
}

type fakeService struct {
	list         func(context.Context, uuid.UUID) ([]exchangemodel.Details, error)
	get          func(context.Context, uuid.UUID, uuid.UUID) (exchangemodel.Details, error)
	confirm      func(context.Context, uuid.UUID, uuid.UUID) error
	decline      func(context.Context, uuid.UUID, uuid.UUID) error
	complete     func(context.Context, uuid.UUID, uuid.UUID) error
	postMessage  func(context.Context, uuid.UUID, uuid.UUID, string) (exchangemodel.Message, error)
	listMessages func(context.Context, uuid.UUID, uuid.UUID) ([]exchangemodel.Message, error)
}

func (f *fakeService) PostMessage(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
	body string,
) (exchangemodel.Message, error) {
	if f.postMessage == nil {
		return exchangemodel.Message{}, nil
	}
	return f.postMessage(ctx, exchangeID, userID, body)
}

func (f *fakeService) ListMessages(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) ([]exchangemodel.Message, error) {
	if f.listMessages == nil {
		return nil, nil
	}
	return f.listMessages(ctx, exchangeID, userID)
}

func (f *fakeService) ListForUser(
	ctx context.Context,
	userID uuid.UUID,
) ([]exchangemodel.Details, error) {
	return f.list(ctx, userID)
}

func (f *fakeService) GetForUser(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) (exchangemodel.Details, error) {
	return f.get(ctx, exchangeID, userID)
}

func (f *fakeService) ConfirmParticipation(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) error {
	if f.confirm == nil {
		return nil
	}
	return f.confirm(ctx, exchangeID, userID)
}

func (f *fakeService) DeclineParticipation(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) error {
	if f.decline == nil {
		return nil
	}
	return f.decline(ctx, exchangeID, userID)
}

func (f *fakeService) CompleteParticipation(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) error {
	if f.complete == nil {
		return nil
	}
	return f.complete(ctx, exchangeID, userID)
}

func performRequest(
	service Service,
	method string,
	path string,
	requireAuth func(http.Handler) http.Handler,
) *httptest.ResponseRecorder {
	return performRequestWithBody(service, method, path, nil, requireAuth)
}

func performRequestWithBody(
	service Service,
	method string,
	path string,
	body io.Reader,
	requireAuth func(http.Handler) http.Handler,
) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	New(service).RegisterRoutes(router, requireAuth)

	request := httptest.NewRequest(method, path, body)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func authenticateAs(userID uuid.UUID) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(authcontext.WithUserID(r.Context(), userID)))
		})
	}
}

func passThroughAuth(next http.Handler) http.Handler {
	return next
}

func exchangeDetails(userID uuid.UUID) exchangemodel.Details {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	return exchangemodel.Details{
		ID:        uuid.New(),
		Status:    "proposed",
		CreatedAt: now,
		UpdatedAt: now,
		Participants: []exchangemodel.DetailsParticipant{{
			User: exchangemodel.ParticipantUser{
				ID:       userID,
				Nickname: "samir",
			},
			GivesItem: exchangemodel.ParticipantItem{
				ID:     uuid.New(),
				Title:  "Book",
				Status: "available",
				Category: exchangemodel.ParticipantCategory{
					Slug: "books",
					Name: "Books",
				},
			},
			ReceivesItem: exchangemodel.ParticipantItem{
				ID:     uuid.New(),
				Title:  "Game",
				Status: "available",
				Category: exchangemodel.ParticipantCategory{
					Slug: "hobby",
					Name: "Hobby",
				},
			},
			Position: 0,
			Status:   "pending",
		}},
	}
}
