package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	exchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/service"
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
	// Обе приходят из обмена: пункт вещи пишет его транзакция, она же и решает, что
	// пункта нет или вещь уже вне оборота.
	ErrUnknownPickupPoint = exchangeservice.ErrUnknownPickupPoint
	ErrConflict           = exchangeservice.ErrConflict
)

type Repository interface {
	Create(context.Context, itemmodel.NewItem) (itemmodel.Item, error)
	GetByID(context.Context, uuid.UUID) (itemmodel.Item, error)
	ListByOwner(context.Context, uuid.UUID) ([]itemmodel.Item, error)
	Update(context.Context, uuid.UUID, itemmodel.Changes) (itemmodel.Item, error)
	Delete(context.Context, uuid.UUID) error
	ListCategories(context.Context) ([]itemmodel.Category, error)
	HasOpenExchange(context.Context, uuid.UUID) (bool, error)
	ClearPickupPoint(context.Context, uuid.UUID, uuid.UUID) error
}

type ExchangeFinder interface {
	FindAndSave(context.Context, exchangemodel.Node) (exchangemodel.SearchResult, error)
	RecordItemPickup(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
}

type errorLogger interface {
	Printf(string, ...any)
}

type Service struct {
	repository Repository
	exchanges  ExchangeFinder
	logger     errorLogger
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

func New(repository Repository, exchangeFinders ...ExchangeFinder) *Service {
	var exchanges ExchangeFinder
	if len(exchangeFinders) > 0 {
		exchanges = exchangeFinders[0]
	}

	return newWithDependencies(repository, exchanges, log.Default())
}

func newWithDependencies(
	repository Repository,
	exchanges ExchangeFinder,
	logger errorLogger,
) *Service {
	return &Service{
		repository: repository,
		exchanges:  exchanges,
		logger:     logger,
	}
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

	created, err := s.repository.Create(ctx, itemmodel.NewItem{
		OwnerID:     input.OwnerID,
		Category:    category,
		Title:       title,
		Description: strings.TrimSpace(input.Description),
		PhotoURLs:   photoURLs,
		Wants:       wants,
	})
	if err != nil {
		return itemmodel.Item{}, err
	}

	s.findExchange(ctx, created)
	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (itemmodel.Item, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]itemmodel.Item, error) {
	return s.repository.ListByOwner(ctx, ownerID)
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

	current, err := s.requireOwner(ctx, id, userID)
	if err != nil {
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

	// Именно изменение значения, а не наличие поля в теле: клиент присылает объявление
	// целиком, и правка одного заголовка иначе выглядела бы как смена условий обмена.
	compatibilityChanged := (changes.Category != nil && *changes.Category != current.Category) ||
		(changes.Wants != nil && !sameWants(changes.Wants, current.Wants))
	if compatibilityChanged {
		hasOpenExchange, err := s.repository.HasOpenExchange(ctx, id)
		if err != nil {
			return itemmodel.Item{}, fmt.Errorf("check item before compatibility update: %w", err)
		}
		if hasOpenExchange {
			return itemmodel.Item{}, ErrItemInChain
		}
	}

	updated, err := s.repository.Update(ctx, id, changes)
	if err != nil {
		return itemmodel.Item{}, err
	}

	if compatibilityChanged {
		s.findExchange(ctx, updated)
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if _, err := s.requireOwner(ctx, id, userID); err != nil {
		return err
	}

	return s.repository.Delete(ctx, id)
}

// SetPickupPoint отмечает, что владелец отнёс вещь в пункт выдачи. Отнести можно и вещь,
// которая ни в каком обмене не участвует: к моменту сборки цепочки она уже будет на месте.
// Саму запись делает exchange — там же, где живёт переход обмена в доставку.
func (s *Service) SetPickupPoint(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
	pickupPointID uuid.UUID,
) error {
	if _, err := s.requireOwner(ctx, id, userID); err != nil {
		return err
	}

	return s.exchanges.RecordItemPickup(ctx, id, userID, pickupPointID)
}

// ClearPickupPoint забирает вещь из пункта домой. Пока вещь занята в обмене, забирать её
// нельзя: остальные участники уже рассчитывают на то, что она лежит на месте.
func (s *Service) ClearPickupPoint(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if _, err := s.requireOwner(ctx, id, userID); err != nil {
		return err
	}

	hasOpenExchange, err := s.repository.HasOpenExchange(ctx, id)
	if err != nil {
		return fmt.Errorf("check item before pickup point reset: %w", err)
	}
	if hasOpenExchange {
		return ErrItemInChain
	}

	return s.repository.ClearPickupPoint(ctx, id, userID)
}

func (s *Service) ListCategories(ctx context.Context) ([]itemmodel.Category, error) {
	return s.repository.ListCategories(ctx)
}

// Владельца проверяем чтением, а не условием в UPDATE: иначе «не моя вещь» и «нет такой
// вещи» слились бы в один результат, и 403 стал бы неотличим от 404.
func (s *Service) requireOwner(ctx context.Context, id uuid.UUID, userID uuid.UUID) (itemmodel.Item, error) {
	item, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return itemmodel.Item{}, err
	}

	if item.OwnerID != userID {
		return itemmodel.Item{}, ErrForbidden
	}

	return item, nil
}

// Желания сравниваются как множества: из БД они приходят отсортированными по слагу,
// а от клиента — в том порядке, в каком он их отметил.
func sameWants(a []string, b []string) bool {
	return slices.Equal(slices.Sorted(slices.Values(a)), slices.Sorted(slices.Values(b)))
}

func (s *Service) findExchange(ctx context.Context, item itemmodel.Item) {
	if s.exchanges == nil {
		return
	}

	_, err := s.exchanges.FindAndSave(ctx, exchangemodel.Node{
		ItemID:  item.ID,
		OwnerID: item.OwnerID,
	})
	if err != nil {
		s.logger.Printf("automatic exchange search for item %s: %v", item.ID, err)
	}
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
