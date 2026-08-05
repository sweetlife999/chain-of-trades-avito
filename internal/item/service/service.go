package service

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	itemmodel "github.com/sweetlife999/chain-of-trades-avito/internal/item/model"
	itemrepository "github.com/sweetlife999/chain-of-trades-avito/internal/item/repository"
)

const (
	maxTitleLength = 120
	maxPhotos      = 10
	maxWants       = 10
)

var (
	ErrValidation      = errors.New("validation error")
	ErrForbidden       = errors.New("item belongs to another user")
	ErrNotFound        = itemrepository.ErrNotFound
	ErrUnknownCategory = itemrepository.ErrUnknownCategory
	ErrItemInChain     = itemrepository.ErrItemInChain
)

type Repository interface {
	Create(context.Context, itemmodel.NewItem) (itemmodel.Item, error)
	GetByID(context.Context, uuid.UUID) (itemmodel.Item, error)
	Update(context.Context, uuid.UUID, itemmodel.Changes) (itemmodel.Item, error)
	Delete(context.Context, uuid.UUID) error
	ListCategories(context.Context) ([]itemmodel.Category, error)
}

type Service struct {
	repository Repository
}

type CreateInput struct {
	OwnerID     uuid.UUID
	Category    string
	Title       string
	Description string
	PhotoURLs   []string
	Wants       []string
}

// Списки различают nil («поле не передали») и пустой срез («передали пустым»):
// второе — ошибка, иначе объявление осталось бы без фото или без желаний.
type UpdateInput struct {
	Category    *string
	Title       *string
	Description *string
	PhotoURLs   []string
	Wants       []string
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

func New(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (itemmodel.Item, error) {
	title := strings.TrimSpace(input.Title)
	if err := validateTitle(title); err != nil {
		return itemmodel.Item{}, err
	}

	category := strings.TrimSpace(input.Category)
	if category == "" {
		return itemmodel.Item{}, validationError("category is required")
	}

	photoURLs, err := cleanPhotoURLs(input.PhotoURLs)
	if err != nil {
		return itemmodel.Item{}, err
	}

	wants, err := cleanWants(input.Wants)
	if err != nil {
		return itemmodel.Item{}, err
	}

	return s.repository.Create(ctx, itemmodel.NewItem{
		OwnerID:     input.OwnerID,
		Category:    category,
		Title:       title,
		Description: strings.TrimSpace(input.Description),
		PhotoURLs:   photoURLs,
		Wants:       wants,
	})
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (itemmodel.Item, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) Update(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
	input UpdateInput,
) (itemmodel.Item, error) {
	if input.Category == nil && input.Title == nil && input.Description == nil &&
		input.PhotoURLs == nil && input.Wants == nil {
		return itemmodel.Item{}, validationError("at least one field must be provided")
	}

	if err := s.requireOwner(ctx, id, userID); err != nil {
		return itemmodel.Item{}, err
	}

	var changes itemmodel.Changes

	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		changes.Description = &description
	}

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if err := validateTitle(title); err != nil {
			return itemmodel.Item{}, err
		}
		changes.Title = &title
	}

	if input.Category != nil {
		category := strings.TrimSpace(*input.Category)
		if category == "" {
			return itemmodel.Item{}, validationError("category is required")
		}
		changes.Category = &category
	}

	if input.PhotoURLs != nil {
		photoURLs, err := cleanPhotoURLs(input.PhotoURLs)
		if err != nil {
			return itemmodel.Item{}, err
		}
		changes.PhotoURLs = photoURLs
	}

	if input.Wants != nil {
		wants, err := cleanWants(input.Wants)
		if err != nil {
			return itemmodel.Item{}, err
		}
		changes.Wants = wants
	}

	return s.repository.Update(ctx, id, changes)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if err := s.requireOwner(ctx, id, userID); err != nil {
		return err
	}

	return s.repository.Delete(ctx, id)
}

func (s *Service) ListCategories(ctx context.Context) ([]itemmodel.Category, error) {
	return s.repository.ListCategories(ctx)
}

// Владельца проверяем чтением, а не условием в UPDATE: иначе «не моя вещь» и «нет такой
// вещи» слились бы в один результат, и 403 стал бы неотличим от 404.
func (s *Service) requireOwner(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	item, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if item.OwnerID != userID {
		return ErrForbidden
	}

	return nil
}

func validateTitle(title string) error {
	length := utf8.RuneCountInString(title)
	if length < 1 || length > maxTitleLength {
		return validationError("title must contain from 1 to 120 characters")
	}

	return nil
}

// Фотографии хранятся ссылками, поэтому строка обязана быть абсолютным http(s)-адресом:
// относительный путь или mailto: в карточке останутся битой картинкой.
func cleanPhotoURLs(photoURLs []string) ([]string, error) {
	cleaned := make([]string, 0, len(photoURLs))
	for _, photoURL := range photoURLs {
		photoURL = strings.TrimSpace(photoURL)
		if photoURL == "" {
			continue
		}

		parsed, err := url.Parse(photoURL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return nil, validationError("photo url must be an absolute http(s) link")
		}

		cleaned = append(cleaned, photoURL)
	}

	if len(cleaned) == 0 {
		return nil, validationError("item must have at least one photo")
	}
	if len(cleaned) > maxPhotos {
		return nil, validationError("item must have at most 10 photos")
	}

	return cleaned, nil
}

// Дубли отбрасываем здесь: в БД пара (item_id, category_id) — первичный ключ, и
// повторённый слаг превратился бы в 409 на ровном месте.
func cleanWants(wants []string) ([]string, error) {
	cleaned := make([]string, 0, len(wants))
	seen := make(map[string]struct{}, len(wants))

	for _, want := range wants {
		want = strings.ToLower(strings.TrimSpace(want))
		if want == "" {
			continue
		}
		if _, duplicate := seen[want]; duplicate {
			continue
		}

		seen[want] = struct{}{}
		cleaned = append(cleaned, want)
	}

	if len(cleaned) == 0 {
		return nil, validationError("item must want at least one category")
	}
	if len(cleaned) > maxWants {
		return nil, validationError("item must want at most 10 categories")
	}

	return cleaned, nil
}

func validationError(message string) error {
	return &ValidationError{message: message}
}
