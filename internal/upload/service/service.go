package service

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// URLPrefix — и маршрут, по которому браузер забирает файл, и то, что лежит в БД.
// Путь относительный: сменится домен — данные не протухнут, и переменной с публичным
// адресом сервиса заводить не нужно.
const URLPrefix = "/uploads/"

const (
	// MaxFileBytes ограничивает картинку, а не тело запроса: границы multipart считает хэндлер.
	MaxFileBytes = 5 << 20
	// http.DetectContentType смотрит максимум на столько байт.
	sniffLength = 512

	directoryMode = 0o755
	fileMode      = 0o644
)

var ErrValidation = errors.New("validation error")

// Расширение берём из содержимого, а не из имени файла: имя приходит из браузера и не
// гарантирует ничего, а под видом .jpg легко приезжает что угодно.
var extensionsByContentType = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type ValidationError struct {
	message string
}

func (e *ValidationError) Error() string {
	return e.message
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}

type Service struct {
	directory string
}

// New заодно создаёт каталог: не хватает прав — узнаем на старте, а не на первой загрузке
// от живого пользователя.
func New(directory string) (*Service, error) {
	if err := os.MkdirAll(directory, directoryMode); err != nil {
		return nil, fmt.Errorf("create uploads directory: %w", err)
	}

	return &Service{directory: directory}, nil
}

func (s *Service) Directory() string {
	return s.directory
}

// Save кладёт картинку на диск и возвращает ссылку на неё.
//
// ponytail: файлы никогда не удаляются — загрузил и передумал, удалил объявление, сменил
// аватарку, файл остался. После даунскейла на клиенте это 200-400 КБ за штуку, диск
// кончится не в этой жизни. Начнёт мешать — команда, которая удаляет файлы старше суток,
// отсутствующие в items.photo_urls и users.photo_url.
func (s *Service) Save(source io.Reader) (string, error) {
	header := make([]byte, sniffLength)
	read, err := io.ReadFull(source, header)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", fmt.Errorf("read uploaded file: %w", err)
	}
	header = header[:read]

	extension, ok := extensionsByContentType[http.DetectContentType(header)]
	if !ok {
		return "", &ValidationError{message: "file must be a jpeg, png or webp image"}
	}

	name := uuid.NewString() + extension
	path := filepath.Join(s.directory, name)

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if err != nil {
		return "", fmt.Errorf("create uploaded file: %w", err)
	}

	// Лишний байт сверх лимита нужен, чтобы отличить «ровно 5 МБ» от «обрезали на 5 МБ».
	rest := io.LimitReader(source, MaxFileBytes-int64(len(header))+1)
	written, copyErr := io.Copy(file, io.MultiReader(bytes.NewReader(header), rest))
	closeErr := file.Close()

	switch {
	case copyErr != nil:
		_ = os.Remove(path)
		return "", fmt.Errorf("write uploaded file: %w", copyErr)
	case closeErr != nil:
		_ = os.Remove(path)
		return "", fmt.Errorf("close uploaded file: %w", closeErr)
	case written > MaxFileBytes:
		_ = os.Remove(path)
		return "", &ValidationError{message: "file must not exceed 5 megabytes"}
	}

	return URLPrefix + name, nil
}

// IsPath отвечает, наша ли это ссылка. Проверка строгая: после префикса ровно одно имя
// файла. Иначе «/uploads/../../etc/passwd» и протокол-относительный «//evil.com/x.jpg»
// проезжали бы как свои — первый мимо каталога, второй мимо префикса вообще.
func IsPath(value string) bool {
	name, found := strings.CutPrefix(value, URLPrefix)

	return found && name != "" && name != "." && name != ".." && !strings.Contains(name, "/")
}
