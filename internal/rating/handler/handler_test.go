package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	ratingdto "github.com/sweetlife999/chain-of-trades-avito/internal/rating/dto"
	ratingmodel "github.com/sweetlife999/chain-of-trades-avito/internal/rating/model"
	ratingservice "github.com/sweetlife999/chain-of-trades-avito/internal/rating/service"
)

type fakeService struct {
	rating     ratingmodel.Rating
	page       ratingmodel.Page
	err        error
	lastRater  uuid.UUID
	lastRated  uuid.UUID
	lastScore  int32
	lastLimit  int32
	lastOffset int32
}

func (f *fakeService) Rate(
	_ context.Context,
	_ uuid.UUID,
	raterID uuid.UUID,
	score int32,
	_ string,
) (ratingmodel.Rating, error) {
	f.lastRater = raterID
	f.lastScore = score
	return f.rating, f.err
}

func (f *fakeService) ListForUser(
	_ context.Context,
	ratedID uuid.UUID,
	limit int32,
	offset int32,
) (ratingmodel.Page, error) {
	f.lastRated = ratedID
	f.lastLimit = limit
	f.lastOffset = offset
	return f.page, f.err
}

func ratingRouter(service Service, userID *uuid.UUID) http.Handler {
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

func do(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestRateReturnsStoredRating(t *testing.T) {
	t.Parallel()

	rated := uuid.New()
	user := uuid.New()
	service := &fakeService{rating: ratingmodel.Rating{RatedUserID: rated, Score: 4, Comment: "ок"}}

	response := do(ratingRouter(service, &user), http.MethodPut,
		"/exchanges/"+uuid.New().String()+"/rating", `{"score":4,"comment":"ок"}`)

	if response.Code != http.StatusOK {
		t.Fatalf("код ответа %d, ожидали 200: %s", response.Code, response.Body.String())
	}
	var decoded ratingdto.RatingResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("ответ не разобрался: %v", err)
	}
	if decoded.RatedUserID != rated.String() || decoded.Score != 4 {
		t.Fatalf("ответ вернулся не тем: %#v", decoded)
	}
	if service.lastRater != user {
		t.Fatal("оценка ушла от чужого имени")
	}
}

// Оценивающий берётся из cookie, а не из тела: иначе клиент оценил бы за другого.
func TestRateRequiresAuthentication(t *testing.T) {
	t.Parallel()

	response := do(ratingRouter(&fakeService{}, nil), http.MethodPut,
		"/exchanges/"+uuid.New().String()+"/rating", `{"score":4}`)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("код ответа %d, ожидали 401", response.Code)
	}
}

func TestRateRejectsBadRequests(t *testing.T) {
	t.Parallel()

	user := uuid.New()
	handler := ratingRouter(&fakeService{}, &user)
	exchange := uuid.New().String()

	cases := map[string]struct{ path, body string }{
		"обмен не UUID": {"/exchanges/not-a-uuid/rating", `{"score":4}`},
		"тело не JSON":  {"/exchanges/" + exchange + "/rating", `score=4`},
		"два объекта":   {"/exchanges/" + exchange + "/rating", `{"score":4}{"score":5}`},
	}

	for name, testCase := range cases {
		response := do(handler, http.MethodPut, testCase.path, testCase.body)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s: код ответа %d, ожидали 400", name, response.Code)
		}
	}
}

// Кого оценивают, приходит из цепочки. Поле в теле — попытка оценить кого-то другого,
// и DisallowUnknownFields обязан её отбить, а не молча проглотить.
func TestRateIgnoresNothingSilently(t *testing.T) {
	t.Parallel()

	user := uuid.New()
	service := &fakeService{}

	response := do(ratingRouter(service, &user), http.MethodPut,
		"/exchanges/"+uuid.New().String()+"/rating", `{"score":5,"rated_user_id":"`+uuid.New().String()+`"}`)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("код ответа %d, ожидали 400", response.Code)
	}
	if service.lastScore != 0 {
		t.Fatal("запрос с лишним полем дошёл до сервиса")
	}
}

func TestRateMapsServiceErrors(t *testing.T) {
	t.Parallel()

	user := uuid.New()
	cases := map[error]int{
		ratingservice.ErrForbidden:    http.StatusForbidden,
		ratingservice.ErrNotFound:     http.StatusNotFound,
		ratingservice.ErrNotCompleted: http.StatusConflict,
		ratingservice.ErrWindowClosed: http.StatusConflict,
	}

	for serviceError, want := range cases {
		handler := ratingRouter(&fakeService{err: serviceError}, &user)
		response := do(handler, http.MethodPut,
			"/exchanges/"+uuid.New().String()+"/rating", `{"score":4}`)
		if response.Code != want {
			t.Fatalf("%v: код ответа %d, ожидали %d", serviceError, response.Code, want)
		}
	}
}

// Лента публична: профиль и так открыт без входа, а рейтинг нужен до решения меняться.
func TestListIsPublic(t *testing.T) {
	t.Parallel()

	service := &fakeService{page: ratingmodel.Page{
		Ratings: []ratingmodel.ReceivedRating{{Score: 5, Comment: "спасибо"}},
		Limit:   ratingservice.DefaultLimit,
	}}
	rated := uuid.New()

	response := do(ratingRouter(service, nil), http.MethodGet, "/users/"+rated.String()+"/ratings", "")

	if response.Code != http.StatusOK {
		t.Fatalf("код ответа %d, ожидали 200: %s", response.Code, response.Body.String())
	}
	if service.lastRated != rated {
		t.Fatal("лента запрошена не у того пользователя")
	}
	if service.lastLimit != ratingservice.DefaultLimit || service.lastOffset != 0 {
		t.Fatalf("умолчания страницы разъехались: limit=%d offset=%d",
			service.lastLimit, service.lastOffset)
	}
}

// В ленте не должно быть ни автора, ни идентификатора отзыва: по любому из них автор
// вычисляется обратно, и анонимность держится именно тем, что их неоткуда взять.
func TestListResponseCarriesNoAuthor(t *testing.T) {
	t.Parallel()

	service := &fakeService{page: ratingmodel.Page{
		Ratings: []ratingmodel.ReceivedRating{{Score: 5, Comment: "спасибо"}},
	}}

	response := do(ratingRouter(service, nil), http.MethodGet,
		"/users/"+uuid.New().String()+"/ratings", "")

	body := response.Body.String()
	for _, forbidden := range []string{"rater", "author", "user_id", "chain", "exchange", "updated_at", `"id"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("в ленте нашлось %q: %s", forbidden, body)
		}
	}
}

func TestListRejectsBadPagination(t *testing.T) {
	t.Parallel()

	handler := ratingRouter(&fakeService{}, nil)
	rated := uuid.New().String()

	for _, path := range []string{
		"/users/not-a-uuid/ratings",
		"/users/" + rated + "/ratings?limit=many",
		"/users/" + rated + "/ratings?offset=first",
	} {
		if response := do(handler, http.MethodGet, path, ""); response.Code != http.StatusBadRequest {
			t.Fatalf("%s: код ответа %d, ожидали 400", path, response.Code)
		}
	}
}
