package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	itemassistant "github.com/sweetlife999/chain-of-trades-avito/internal/itemassistant/service"
)

type fakeService struct {
	submittedOwner uuid.UUID
	submittedInput string
	job            itemassistant.Job
	err            error
}

func (f *fakeService) Submit(ownerID uuid.UUID, input string) (itemassistant.Job, error) {
	f.submittedOwner = ownerID
	f.submittedInput = input
	return f.job, f.err
}

func (f *fakeService) Get(ownerID, id uuid.UUID) (itemassistant.Job, error) {
	f.submittedOwner = ownerID
	return f.job, f.err
}

func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(authcontext.WithUserID(r.Context(), uuid.MustParse("55f750bb-b53a-4f87-a2a8-5ad1952d9150"))))
	})
}

func TestSubmitReturnsAcceptedJob(t *testing.T) {
	jobID := uuid.New()
	service := &fakeService{job: itemassistant.Job{
		ID: jobID, Status: itemassistant.StatusPending,
		CreatedAt: time.Date(2026, time.August, 14, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.August, 14, 0, 0, 0, 0, time.UTC),
	}}
	router := chi.NewRouter()
	New(service).RegisterRoutes(router, requireAuth)

	request := httptest.NewRequest(http.MethodPost, "/items/ai-suggestions", strings.NewReader(`{"input":"красный горный велосипед"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status: got %d, body %s", response.Code, response.Body.String())
	}
	if service.submittedInput != "красный горный велосипед" {
		t.Fatalf("input: %q", service.submittedInput)
	}
	if !strings.Contains(response.Body.String(), jobID.String()) || !strings.Contains(response.Body.String(), `"pending"`) {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
}

func TestGetDoesNotExposeOtherUsersJob(t *testing.T) {
	service := &fakeService{err: itemassistant.ErrForbidden}
	router := chi.NewRouter()
	New(service).RegisterRoutes(router, requireAuth)
	request := httptest.NewRequest(http.MethodGet, "/items/ai-suggestions/"+uuid.NewString(), nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, body %s", response.Code, response.Body.String())
	}
}

func TestDisabledAssistantReturnsServiceUnavailable(t *testing.T) {
	router := chi.NewRouter()
	New(nil).RegisterRoutes(router, requireAuth)
	request := httptest.NewRequest(http.MethodPost, "/items/ai-suggestions", strings.NewReader(`{"input":"красный горный велосипед"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: got %d, body %s", response.Code, response.Body.String())
	}
}
