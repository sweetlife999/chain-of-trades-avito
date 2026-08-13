package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	uploaddto "github.com/sweetlife999/chain-of-trades-avito/internal/upload/dto"
	uploadservice "github.com/sweetlife999/chain-of-trades-avito/internal/upload/service"
)

// Запас поверх лимита самой картинки — на границы multipart и заголовки частей.
const maxRequestBodyBytes = uploadservice.MaxFileBytes + 32<<10

type Service interface {
	Save(io.Reader) (string, error)
	Directory() string
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router chi.Router, requireAuth func(http.Handler) http.Handler) {
	router.With(requireAuth).Post("/uploads", h.upload)

	// Отдаём без авторизации: карточка вещи и профиль публичны, а значит публичны и их
	// фотографии. HEAD регистрируем рядом с GET: FileServer его умеет, а без маршрута
	// chi отвечал бы 405 на файл, который прекрасно отдаётся по GET.
	files := h.files()
	router.Get(uploadservice.URLPrefix+"*", files.ServeHTTP)
	router.Head(uploadservice.URLPrefix+"*", files.ServeHTTP)
}

// @Summary     Загрузить фотографию
// @Description Требует cookie `access_token`. Принимает один файл в поле `file` формы
// @Description multipart/form-data: jpeg, png или webp размером до 5 МБ. Тип определяется по
// @Description содержимому файла, а имя присваивает сервер, поэтому расширение в запросе ни на
// @Description что не влияет.
// @Description
// @Description Возвращает ссылку, которую нужно передать в `photo_urls` объявления или в
// @Description `photo_url` профиля. Сама по себе загрузка ни к чему файл не привязывает.
// @Tags        uploads
// @Accept      mpfd
// @Produce     json
// @Param       file formData file true "Файл изображения"
// @Success     201  {object} uploaddto.UploadResponse "Загружено, ссылка в поле url"
// @Failure     400  {object} uploaddto.UploadError    "Нет файла в запросе или это не картинка поддерживаемого формата"
// @Failure     401  {object} uploaddto.UploadError    "Нет или истекла cookie access_token"
// @Failure     413  {object} uploaddto.UploadError    "Файл больше 5 МБ"
// @Failure     500  {object} uploaddto.UploadError    "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /uploads [post]
func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	if _, ok := authcontext.UserID(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	// Тело такого размера multipart держит в памяти целиком, но временный файл он всё же
	// может завести — например на разборе битой формы. Убираем за собой в любом случае.
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeParseError(w, err)
		return
	}
	defer file.Close()

	url, err := h.service.Save(file)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, uploaddto.UploadResponse{URL: url})
}

// Раздаём файлы сами, а не Caddy: один и тот же маршрут работает и в разработке, и за
// прокси. Склейкой пути занимаются StripPrefix и FileServer, поэтому вылезти из каталога
// запросом нельзя — своей арифметики с путями здесь нет.
func (h *Handler) files() http.Handler {
	files := http.FileServer(http.Dir(h.service.Directory()))

	return http.StripPrefix(strings.TrimSuffix(uploadservice.URLPrefix, "/"),
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// FileServer на каталоге отдаёт листинг, а это ссылки на все фотографии сервиса разом.
			if strings.HasSuffix(r.URL.Path, "/") {
				http.NotFound(w, r)
				return
			}

			// Имя файла — свежий uuid, содержимое по ссылке не меняется никогда.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			files.ServeHTTP(w, r)
		}))
}

// Перебор по размеру ловится ещё до сервиса, поэтому отдельный ответ на него живёт здесь.
func writeParseError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "file must not exceed 5 megabytes")
		return
	}

	writeError(w, http.StatusBadRequest, "request must contain a file field")
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, uploadservice.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("upload handler: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, uploaddto.UploadError{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode JSON response: %v", err)
	}
}
