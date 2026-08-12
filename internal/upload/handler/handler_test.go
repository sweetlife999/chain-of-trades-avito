package handler

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
)

type fakeService struct {
	save      func(io.Reader) (string, error)
	directory string
}

func (f *fakeService) Save(source io.Reader) (string, error) {
	return f.save(source)
}

func (f *fakeService) Directory() string {
	return f.directory
}

func TestUploadReturns201WithURL(t *testing.T) {
	t.Parallel()

	var received []byte
	service := &fakeService{
		save: func(source io.Reader) (string, error) {
			content, err := io.ReadAll(source)
			received = content

			return "/uploads/photo.png", err
		},
	}

	content := encodePNG(t)
	contentType, body := multipartBody(t, "file", "anything.txt", content)

	response := performRequest(service, http.MethodPost, "/uploads", contentType, body, authenticateAs(uuid.New()))
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var decoded struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded.URL != "/uploads/photo.png" {
		t.Fatalf("url = %q", decoded.URL)
	}
	if !bytes.Equal(received, content) {
		t.Fatalf("сервис получил %d байт, ожидалось %d", len(received), len(content))
	}
}

// Свободная загрузка превратила бы сервис в бесплатный файлохостинг для кого угодно.
func TestUploadRequiresAuthentication(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		save: func(io.Reader) (string, error) {
			t.Fatal("Save() не должен вызываться без пользователя")
			return "", nil
		},
	}

	contentType, body := multipartBody(t, "file", "photo.png", encodePNG(t))

	response := performRequest(service, http.MethodPost, "/uploads", contentType, body, passThroughAuth)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestUploadRejectsRequestWithoutFile(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		save: func(io.Reader) (string, error) {
			t.Fatal("Save() не должен вызываться без файла в запросе")
			return "", nil
		},
	}

	cases := []struct {
		name        string
		contentType string
		body        io.Reader
	}{
		{name: "не multipart", contentType: "application/json", body: strings.NewReader(`{"file":"photo.png"}`)},
		{name: "другое поле формы", contentType: "", body: nil},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			contentType, body := testCase.contentType, testCase.body
			if body == nil {
				contentType, body = multipartBody(t, "photo", "photo.png", encodePNG(t))
			}

			response := performRequest(service, http.MethodPost, "/uploads", contentType, body, authenticateAs(uuid.New()))
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestUploadRejectsOversizedBody(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		save: func(io.Reader) (string, error) {
			t.Fatal("Save() не должен вызываться: тело обрывается раньше")
			return "", nil
		},
	}

	contentType, body := multipartBody(t, "file", "photo.png", make([]byte, maxRequestBodyBytes+1))

	response := performRequest(service, http.MethodPost, "/uploads", contentType, body, authenticateAs(uuid.New()))
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusRequestEntityTooLarge, response.Body.String())
	}
}

func TestFilesServeSavedPhoto(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	content := encodePNG(t)
	if err := os.WriteFile(filepath.Join(directory, "photo.png"), content, 0o600); err != nil {
		t.Fatalf("write photo: %v", err)
	}

	response := performRequest(&fakeService{directory: directory}, http.MethodGet, "/uploads/photo.png", "", nil, passThroughAuth)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !bytes.Equal(response.Body.Bytes(), content) {
		t.Fatalf("отдано %d байт, ожидалось %d", response.Body.Len(), len(content))
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", response.Header().Get("X-Content-Type-Options"))
	}

	head := performRequest(&fakeService{directory: directory}, http.MethodHead, "/uploads/photo.png", "", nil, passThroughAuth)
	if head.Code != http.StatusOK {
		t.Fatalf("HEAD status = %d, want %d", head.Code, http.StatusOK)
	}
}

// Листинг каталога отдал бы ссылки на все фотографии сервиса разом.
func TestFilesDoNotListDirectory(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "photo.png"), encodePNG(t), 0o600); err != nil {
		t.Fatalf("write photo: %v", err)
	}

	response := performRequest(&fakeService{directory: directory}, http.MethodGet, "/uploads/", "", nil, passThroughAuth)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusNotFound, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "photo.png") {
		t.Fatalf("в ответе перечислены файлы каталога: %s", response.Body.String())
	}
}

func TestFilesDoNotEscapeTheDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	directory := filepath.Join(root, "uploads")
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatalf("create uploads directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "secret.txt"), []byte("секрет"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	for _, path := range []string{"/uploads/../secret.txt", "/uploads/%2e%2e/secret.txt"} {
		response := performRequest(&fakeService{directory: directory}, http.MethodGet, path, "", nil, passThroughAuth)
		if strings.Contains(response.Body.String(), "секрет") {
			t.Fatalf("%s отдал файл за пределами каталога: %s", path, response.Body.String())
		}
	}
}

func performRequest(
	service Service,
	method string,
	target string,
	contentType string,
	body io.Reader,
	auth func(http.Handler) http.Handler,
) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	New(service).RegisterRoutes(router, auth)

	request := httptest.NewRequest(method, target, body)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	return response
}

func multipartBody(t *testing.T, field string, filename string, content []byte) (string, io.Reader) {
	t.Helper()

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	return writer.FormDataContentType(), &buffer
}

func encodePNG(t *testing.T) []byte {
	t.Helper()

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	return buffer.Bytes()
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
