package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	adminauditmodel "github.com/sweetlife999/chain-of-trades-avito/internal/adminaudit/model"
	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
)

type fakeService struct {
	state   adminauditmodel.UserBlockState
	page    adminauditmodel.Page
	err     error
	adminID uuid.UUID
	userID  uuid.UUID
	filter  adminauditmodel.Filter
}

func (f *fakeService) BlockUser(_ context.Context, adminID, userID uuid.UUID) (adminauditmodel.UserBlockState, error) {
	f.adminID, f.userID = adminID, userID
	return f.state, f.err
}
func (f *fakeService) UnblockUser(_ context.Context, adminID, userID uuid.UUID) (adminauditmodel.UserBlockState, error) {
	f.adminID, f.userID = adminID, userID
	return f.state, f.err
}
func (f *fakeService) List(_ context.Context, filter adminauditmodel.Filter) (adminauditmodel.Page, error) {
	f.filter = filter
	return f.page, f.err
}

func TestBlockUserUsesAuthenticatedAdmin(t *testing.T) {
	t.Parallel()
	adminID, userID := uuid.New(), uuid.New()
	service := &fakeService{state: adminauditmodel.UserBlockState{ID: userID, Nickname: "blocked", IsBlocked: true}}
	router := chi.NewRouter()
	New(service).RegisterRoutes(router)
	request := httptest.NewRequest(http.MethodPost, "/users/"+userID.String()+"/block", nil)
	request = request.WithContext(authcontext.WithUserID(request.Context(), adminID))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if service.adminID != adminID || service.userID != userID {
		t.Fatalf("ids = %s/%s", service.adminID, service.userID)
	}
}

func TestBlockUserRequiresAuthentication(t *testing.T) {
	t.Parallel()
	router := chi.NewRouter()
	New(&fakeService{}).RegisterRoutes(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/users/"+uuid.NewString()+"/block", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestListParsesFilters(t *testing.T) {
	t.Parallel()
	adminID := uuid.New()
	service := &fakeService{page: adminauditmodel.Page{Entries: []adminauditmodel.Entry{}}}
	router := chi.NewRouter()
	New(service).RegisterRoutes(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/audit-log?admin_id="+adminID.String()+"&action=user_blocked&limit=10&offset=2", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if service.filter.AdminID == nil || *service.filter.AdminID != adminID || service.filter.Action != "user_blocked" || service.filter.Limit != 10 || service.filter.Offset != 2 {
		t.Fatalf("filter = %+v", service.filter)
	}
}
