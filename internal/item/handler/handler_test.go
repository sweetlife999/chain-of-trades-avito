package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	itemmodel "github.com/sweetlife999/chain-of-trades-avito/internal/item/model"
	itemservice "github.com/sweetlife999/chain-of-trades-avito/internal/item/service"
)

type fakeService struct {
	create     func(context.Context, itemservice.CreateInput) (itemmodel.Item, error)
	get        func(context.Context, uuid.UUID) (itemmodel.Item, error)
	list       func(context.Context, uuid.UUID) ([]itemmodel.Item, error)
	update     func(context.Context, uuid.UUID, uuid.UUID, itemservice.UpdateInput) (itemmodel.Item, error)
	remove     func(context.Context, uuid.UUID, uuid.UUID) error
	setSearch  func(context.Context, uuid.UUID, uuid.UUID, bool) (itemmodel.Item, error)
	categories func(context.Context) ([]itemmodel.Category, error)

	setPickup   func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	clearPickup func(context.Context, uuid.UUID, uuid.UUID) error
}

func (f *fakeService) SetPickupPoint(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
	pickupPointID uuid.UUID,
) error {
	return f.setPickup(ctx, id, userID, pickupPointID)
}

func (f *fakeService) ClearPickupPoint(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return f.clearPickup(ctx, id, userID)
}

func (f *fakeService) Create(ctx context.Context, input itemservice.CreateInput) (itemmodel.Item, error) {
	return f.create(ctx, input)
}

func (f *fakeService) GetByID(ctx context.Context, id uuid.UUID) (itemmodel.Item, error) {
	return f.get(ctx, id)
}

func (f *fakeService) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]itemmodel.Item, error) {
	return f.list(ctx, ownerID)
}

func (f *fakeService) Update(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
	input itemservice.UpdateInput,
) (itemmodel.Item, error) {
	return f.update(ctx, id, userID, input)
}

func (f *fakeService) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return f.remove(ctx, id, userID)
}

func (f *fakeService) SetSearchVisibility(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
	enabled bool,
) (itemmodel.Item, error) {
	return f.setSearch(ctx, id, userID, enabled)
}

func (f *fakeService) ListCategories(ctx context.Context) ([]itemmodel.Category, error) {
	return f.categories(ctx)
}

const createBody = `{
	"category":"bikes",
	"title":"Велосипед",
	"photo_urls":["https://example.com/1.jpg"],
	"wants":["consoles"]
}`

func TestCreateReturns201WithLocation(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	userID := uuid.New()

	var received itemservice.CreateInput
	service := &fakeService{
		create: func(_ context.Context, input itemservice.CreateInput) (itemmodel.Item, error) {
			received = input
			return itemmodel.Item{ID: id, OwnerID: input.OwnerID}, nil
		},
	}

	response := performRequest(service, http.MethodPost, "/items", createBody, authenticateAs(userID))

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if response.Header().Get("Location") != "/items/"+id.String() {
		t.Fatalf("Location = %q", response.Header().Get("Location"))
	}
	// Владелец берётся из токена: подделать его телом запроса нельзя.
	if received.OwnerID != userID {
		t.Fatalf("owner = %v, want %v", received.OwnerID, userID)
	}
}

func TestGetReturns200(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	service := &fakeService{
		get: func(_ context.Context, actualID uuid.UUID) (itemmodel.Item, error) {
			if actualID != id {
				t.Fatalf("GetByID() id = %v, want %v", actualID, id)
			}
			return itemmodel.Item{ID: id, PhotoURLs: []string{"https://example.com/1.jpg"}}, nil
		},
	}

	response := performRequest(service, http.MethodGet, "/items/"+id.String(), "", passThroughAuth)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestListMineReturns200WithOwnerFromToken(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	service := &fakeService{
		list: func(_ context.Context, ownerID uuid.UUID) ([]itemmodel.Item, error) {
			if ownerID != userID {
				t.Fatalf("ListByOwner() owner = %v, want %v", ownerID, userID)
			}
			return []itemmodel.Item{{ID: uuid.New(), OwnerID: ownerID, Title: "Велосипед"}}, nil
		},
	}

	response := performRequest(service, http.MethodGet, "/items", "", authenticateAs(userID))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"title":"Велосипед"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestListMineReturnsEmptyArrayNotNull(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		list: func(context.Context, uuid.UUID) ([]itemmodel.Item, error) {
			return nil, nil
		},
	}

	response := performRequest(service, http.MethodGet, "/items", "", authenticateAs(uuid.New()))
	if body := strings.TrimSpace(response.Body.String()); body != "[]" {
		t.Fatalf("body = %s, want []", body)
	}
}

func TestUpdateReturns200AndPassesEmptyPhotoList(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	userID := uuid.New()

	var received itemservice.UpdateInput
	service := &fakeService{
		update: func(_ context.Context, actualID uuid.UUID, actualUserID uuid.UUID, input itemservice.UpdateInput) (itemmodel.Item, error) {
			if actualID != id || actualUserID != userID {
				t.Fatalf("Update() id = %v, user = %v", actualID, actualUserID)
			}
			received = input
			return itemmodel.Item{ID: id}, nil
		},
	}

	response := performRequest(service, http.MethodPatch, "/items/"+id.String(),
		`{"photo_urls":[],"title":"Новый"}`, authenticateAs(userID))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	// Пустой список обязан доехать до сервиса непустым указателем на пустоту:
	// именно он превращается в 400, а не в «поле не передали».
	if received.PhotoURLs == nil || len(received.PhotoURLs) != 0 {
		t.Fatalf("photo_urls = %#v, want пустой непустой срез", received.PhotoURLs)
	}
	if received.Wants != nil {
		t.Fatalf("wants = %#v, want nil для непереданного поля", received.Wants)
	}
}

func TestDeleteReturns204(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	userID := uuid.New()
	service := &fakeService{
		remove: func(_ context.Context, actualID uuid.UUID, actualUserID uuid.UUID) error {
			if actualID != id || actualUserID != userID {
				t.Fatalf("Delete() id = %v, user = %v", actualID, actualUserID)
			}
			return nil
		},
	}

	response := performRequest(service, http.MethodDelete, "/items/"+id.String(), "", authenticateAs(userID))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusNoContent, response.Body.String())
	}
	if response.Body.Len() != 0 {
		t.Fatalf("204 с телом: %s", response.Body.String())
	}
}

func TestSetPickupPointReturns204(t *testing.T) {
	t.Parallel()

	id, userID, pointID := uuid.New(), uuid.New(), uuid.New()
	service := &fakeService{
		setPickup: func(_ context.Context, actualID, actualUser, actualPoint uuid.UUID) error {
			if actualID != id || actualUser != userID || actualPoint != pointID {
				t.Fatalf("SetPickupPoint(%v, %v, %v)", actualID, actualUser, actualPoint)
			}
			return nil
		},
	}

	response := performRequest(
		service,
		http.MethodPost,
		"/items/"+id.String()+"/pickup",
		`{"pickup_point_id":"`+pointID.String()+`"}`,
		authenticateAs(userID),
	)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusNoContent, response.Body.String())
	}
}

func TestSetPickupPointRejectsBrokenPoint(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		body       string
		serviceErr error
		wantStatus int
	}{
		"не UUID в теле":   {body: `{"pickup_point_id":"not-a-uuid"}`, wantStatus: http.StatusBadRequest},
		"пункта нет":       {serviceErr: itemservice.ErrUnknownPickupPoint, wantStatus: http.StatusBadRequest},
		"вещь вне оборота": {serviceErr: itemservice.ErrConflict, wantStatus: http.StatusConflict},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			body := test.body
			if body == "" {
				body = `{"pickup_point_id":"` + uuid.New().String() + `"}`
			}
			service := &fakeService{
				setPickup: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
					return test.serviceErr
				},
			}

			response := performRequest(
				service,
				http.MethodPost,
				"/items/"+uuid.New().String()+"/pickup",
				body,
				authenticateAs(uuid.New()),
			)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func TestClearPickupPointReturns409ForItemInChain(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		clearPickup: func(context.Context, uuid.UUID, uuid.UUID) error {
			return itemservice.ErrItemInChain
		},
	}

	response := performRequest(
		service,
		http.MethodDelete,
		"/items/"+uuid.New().String()+"/pickup",
		"",
		authenticateAs(uuid.New()),
	)
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusConflict, response.Body.String())
	}
}

func TestSetSearchVisibilityReturnsUpdatedItem(t *testing.T) {
	t.Parallel()

	itemID, userID := uuid.New(), uuid.New()
	tests := []struct {
		name    string
		method  string
		enabled bool
		status  string
	}{
		{name: "вернуть в поиск", method: http.MethodPut, enabled: true, status: "available"},
		{name: "снять с поиска", method: http.MethodDelete, enabled: false, status: "withdrawn"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := &fakeService{
				setSearch: func(
					_ context.Context,
					actualItemID uuid.UUID,
					actualUserID uuid.UUID,
					enabled bool,
				) (itemmodel.Item, error) {
					if actualItemID != itemID || actualUserID != userID || enabled != test.enabled {
						t.Fatalf("SetSearchVisibility(%s, %s, %t)", actualItemID, actualUserID, enabled)
					}
					return itemmodel.Item{ID: itemID, OwnerID: userID, Status: test.status}, nil
				},
			}

			response := performRequest(
				service,
				test.method,
				"/items/"+itemID.String()+"/search",
				"",
				authenticateAs(userID),
			)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), `"status":"`+test.status+`"`) {
				t.Fatalf("body = %s, want status %s", response.Body.String(), test.status)
			}
		})
	}
}

func TestSetSearchVisibilityErrorStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "чужое объявление", err: itemservice.ErrForbidden, wantStatus: http.StatusForbidden},
		{name: "объявление не найдено", err: itemservice.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "объявление занято", err: itemservice.ErrSearchVisibilityConflict, wantStatus: http.StatusConflict},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := &fakeService{
				setSearch: func(context.Context, uuid.UUID, uuid.UUID, bool) (itemmodel.Item, error) {
					return itemmodel.Item{}, test.err
				},
			}
			response := performRequest(
				service,
				http.MethodDelete,
				"/items/"+uuid.New().String()+"/search",
				"",
				authenticateAs(uuid.New()),
			)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func TestListCategoriesReturns200(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		categories: func(context.Context) ([]itemmodel.Category, error) {
			return []itemmodel.Category{{Slug: "bikes", Name: "Велосипеды и транспорт"}}, nil
		},
	}

	response := performRequest(service, http.MethodGet, "/categories", "", passThroughAuth)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"slug":"bikes"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestProtectedRoutesRequireAuthenticatedUser(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	tests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodPost, path: "/items", body: createBody},
		{method: http.MethodGet, path: "/items"},
		{method: http.MethodPatch, path: "/items/" + id.String(), body: `{"title":"Новый"}`},
		{method: http.MethodDelete, path: "/items/" + id.String()},
		{method: http.MethodPut, path: "/items/" + id.String() + "/search"},
		{method: http.MethodDelete, path: "/items/" + id.String() + "/search"},
		{
			method: http.MethodPost,
			path:   "/items/" + id.String() + "/pickup",
			body:   `{"pickup_point_id":"` + uuid.New().String() + `"}`,
		},
		{method: http.MethodDelete, path: "/items/" + id.String() + "/pickup"},
	}

	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			t.Parallel()

			// Фейк без единой заданной функции: если хэндлер всё же дойдёт до сервиса,
			// тест упадёт на nil-вызове.
			response := performRequest(&fakeService{}, test.method, test.path, test.body, passThroughAuth)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusUnauthorized, response.Body.String())
			}
		})
	}
}

func TestHandlerErrorStatuses(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{name: "битый JSON", method: http.MethodPost, path: "/items", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "неизвестное поле", method: http.MethodPost, path: "/items", body: `{"unknown":true}`, wantStatus: http.StatusBadRequest},
		{name: "не UUID", method: http.MethodGet, path: "/items/not-a-uuid", wantStatus: http.StatusBadRequest},
		{
			name: "не прошло проверку", method: http.MethodPost, path: "/items", body: createBody,
			serviceErr: itemservice.ErrValidation, wantStatus: http.StatusBadRequest,
		},
		{
			name: "неизвестная категория", method: http.MethodPost, path: "/items", body: createBody,
			serviceErr: itemservice.ErrUnknownCategory, wantStatus: http.StatusBadRequest,
		},
		{
			name: "чужое объявление", method: http.MethodPatch, path: "/items/" + id.String(), body: `{"title":"Новый"}`,
			serviceErr: itemservice.ErrForbidden, wantStatus: http.StatusForbidden,
		},
		{
			name: "нет такого объявления", method: http.MethodGet, path: "/items/" + id.String(),
			serviceErr: itemservice.ErrNotFound, wantStatus: http.StatusNotFound,
		},
		{
			name: "вещь в цепочке", method: http.MethodDelete, path: "/items/" + id.String(),
			serviceErr: itemservice.ErrItemInChain, wantStatus: http.StatusConflict,
		},
		{
			name: "внутренняя ошибка", method: http.MethodGet, path: "/items/" + id.String(),
			serviceErr: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := &fakeService{
				create: func(context.Context, itemservice.CreateInput) (itemmodel.Item, error) {
					return itemmodel.Item{}, test.serviceErr
				},
				get: func(context.Context, uuid.UUID) (itemmodel.Item, error) {
					return itemmodel.Item{}, test.serviceErr
				},
				update: func(context.Context, uuid.UUID, uuid.UUID, itemservice.UpdateInput) (itemmodel.Item, error) {
					return itemmodel.Item{}, test.serviceErr
				},
				remove: func(context.Context, uuid.UUID, uuid.UUID) error {
					return test.serviceErr
				},
			}

			response := performRequest(service, test.method, test.path, test.body, authenticateAs(uuid.New()))
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func performRequest(
	service Service,
	method string,
	path string,
	body string,
	requireAuth func(http.Handler) http.Handler,
) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	New(service).RegisterRoutes(router, requireAuth)

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func authenticateAs(userID uuid.UUID) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(authcontext.WithUserID(r.Context(), userID))
			next.ServeHTTP(w, r)
		})
	}
}

func passThroughAuth(next http.Handler) http.Handler {
	return next
}
