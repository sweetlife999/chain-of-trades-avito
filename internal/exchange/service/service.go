package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	exchangerepository "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/repository"
)

const (
	maxParticipants = 5
	// Тред нужен, чтобы договориться о передаче вещей, а не для длинных писем.
	maxMessageLength = 2000
)

var (
	ErrInvalidCycle      = errors.New("invalid exchange cycle")
	ErrValidation        = errors.New("invalid exchange input")
	ErrForbidden         = exchangerepository.ErrNotParticipant
	ErrConflict          = exchangerepository.ErrConflict
	ErrNotFound          = exchangerepository.ErrNotFound
	ErrDuplicateExchange = exchangerepository.ErrDuplicateExchange
)

type Repository interface {
	FindNeighbors(context.Context, uuid.UUID) ([]exchangemodel.Node, error)
	HasUserBlockConflict(context.Context, uuid.UUID, []uuid.UUID) (bool, error)
	SaveExchange(context.Context, exchangemodel.Exchange) (uuid.UUID, error)
	ListByUser(context.Context, uuid.UUID) ([]exchangemodel.Details, error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (exchangemodel.Details, error)
	ConfirmParticipation(context.Context, uuid.UUID, uuid.UUID) error
	DeclineParticipation(context.Context, uuid.UUID, uuid.UUID) ([]exchangemodel.Node, error)
	CompleteParticipation(context.Context, uuid.UUID, uuid.UUID) error
	ExchangeAccess(context.Context, uuid.UUID, uuid.UUID) (string, bool, error)
	CreateMessage(context.Context, uuid.UUID, uuid.UUID, string) (exchangemodel.Message, error)
	ListMessages(context.Context, uuid.UUID) ([]exchangemodel.Message, error)
	MarkMessagesRead(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
}

type Service struct {
	repository Repository
	logger     errorLogger
}

type errorLogger interface {
	Printf(string, ...any)
}

func New(repository Repository) *Service {
	return newWithDependencies(repository, log.Default())
}

func newWithDependencies(repository Repository, logger errorLogger) *Service {
	return &Service{repository: repository, logger: logger}
}

// FindCycle ищет первый обмен, который начинается со start и возвращается в него.
// Отсутствие подходящего обмена — нормальный результат: в этом случае возвращается nil, nil.
func (s *Service) FindCycle(ctx context.Context, start exchangemodel.Node) ([]exchangemodel.Node, error) {
	return s.findCycle(ctx, start, nil)
}

func (s *Service) findCycle(
	ctx context.Context,
	start exchangemodel.Node,
	excludedCycles map[string]struct{},
) ([]exchangemodel.Node, error) {
	path := []exchangemodel.Node{start}
	visitedItems := map[uuid.UUID]struct{}{start.ItemID: {}}
	visitedOwners := map[uuid.UUID]struct{}{start.OwnerID: {}}

	var cycle []exchangemodel.Node

	var dfs func(exchangemodel.Node) (bool, error)
	dfs = func(current exchangemodel.Node) (bool, error) {
		if err := ctx.Err(); err != nil {
			return false, err
		}

		neighbors, err := s.repository.FindNeighbors(ctx, current.ItemID)
		if err != nil {
			return false, fmt.Errorf("find neighbors for item %s: %w", current.ItemID, err)
		}

		for _, next := range neighbors {
			// На глубине 5 ещё можно замкнуть путь, но нельзя добавить шестого участника.
			if next.ItemID == start.ItemID {
				if len(path) >= 2 {
					candidate := append([]exchangemodel.Node(nil), path...)
					if _, excluded := excludedCycles[cycleKey(candidate)]; excluded {
						continue
					}

					cycle = candidate
					return true, nil
				}

				continue
			}

			if len(path) >= maxParticipants {
				continue
			}

			if _, visited := visitedItems[next.ItemID]; visited {
				continue
			}

			if _, visited := visitedOwners[next.OwnerID]; visited {
				continue
			}

			pathOwnerIDs := make([]uuid.UUID, len(path))
			for index, node := range path {
				pathOwnerIDs[index] = node.OwnerID
			}

			blocked, err := s.repository.HasUserBlockConflict(ctx, next.OwnerID, pathOwnerIDs)
			if err != nil {
				return false, fmt.Errorf("check blocks for user %s: %w", next.OwnerID, err)
			}
			if blocked {
				continue
			}

			visitedItems[next.ItemID] = struct{}{}
			visitedOwners[next.OwnerID] = struct{}{}
			path = append(path, next)

			found, err := dfs(next)

			path = path[:len(path)-1]
			delete(visitedItems, next.ItemID)
			delete(visitedOwners, next.OwnerID)

			if err != nil {
				return false, err
			}

			if found {
				return true, nil
			}
		}

		return false, nil
	}

	found, err := dfs(start)
	if err != nil {
		return nil, err
	}

	if !found {
		return nil, nil
	}

	return cycle, nil
}

// SaveCycle переводит найденный путь в участников обмена и сохраняет их одной транзакцией.
func (s *Service) SaveCycle(ctx context.Context, cycle []exchangemodel.Node) (uuid.UUID, error) {
	if err := validateCycle(cycle); err != nil {
		return uuid.Nil, err
	}

	participants := make([]exchangemodel.Participant, len(cycle))
	for index, node := range cycle {
		next := cycle[(index+1)%len(cycle)]
		participants[index] = exchangemodel.Participant{
			UserID:         node.OwnerID,
			GivesItemID:    node.ItemID,
			ReceivesItemID: next.ItemID,
			Position:       int32(index),
		}
	}

	id, err := s.repository.SaveExchange(ctx, exchangemodel.Exchange{Participants: participants})
	if err != nil {
		return uuid.Nil, fmt.Errorf("save cycle: %w", err)
	}

	return id, nil
}

// FindAndSave запускает полный сценарий: ищет обмен от нового объявления и,
// если находит, сохраняет его. Отсутствие обмена не считается ошибкой.
func (s *Service) FindAndSave(
	ctx context.Context,
	start exchangemodel.Node,
) (exchangemodel.SearchResult, error) {
	excludedCycles := make(map[string]struct{})

	for {
		cycle, err := s.findCycle(ctx, start, excludedCycles)
		if err != nil {
			return exchangemodel.SearchResult{}, fmt.Errorf("search exchange: %w", err)
		}
		if cycle == nil {
			return exchangemodel.SearchResult{}, nil
		}

		exchangeID, err := s.SaveCycle(ctx, cycle)
		if errors.Is(err, ErrDuplicateExchange) {
			// Подпись уже могла существовать до запуска поиска или быть сохранена
			// параллельным DFS. Исключаем этот цикл и ищем следующую альтернативу.
			excludedCycles[cycleKey(cycle)] = struct{}{}
			continue
		}
		if err != nil {
			return exchangemodel.SearchResult{}, fmt.Errorf("persist found exchange: %w", err)
		}

		return exchangemodel.SearchResult{
			ExchangeID: exchangeID,
			Found:      true,
		}, nil
	}
}

func (s *Service) ListForUser(ctx context.Context, userID uuid.UUID) ([]exchangemodel.Details, error) {
	exchanges, err := s.repository.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list exchanges for user: %w", err)
	}

	if exchanges == nil {
		return []exchangemodel.Details{}, nil
	}

	return exchanges, nil
}

func (s *Service) GetForUser(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) (exchangemodel.Details, error) {
	exchange, err := s.repository.GetByID(ctx, exchangeID, userID)
	if err != nil {
		return exchangemodel.Details{}, fmt.Errorf("get exchange for user: %w", err)
	}

	for _, participant := range exchange.Participants {
		if participant.User.ID == userID {
			return exchange, nil
		}
	}

	return exchangemodel.Details{}, ErrForbidden
}

func (s *Service) ConfirmParticipation(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) error {
	if err := s.repository.ConfirmParticipation(ctx, exchangeID, userID); err != nil {
		return fmt.Errorf("confirm exchange participation: %w", err)
	}

	return nil
}

func (s *Service) DeclineParticipation(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) error {
	recoveryNodes, err := s.repository.DeclineParticipation(ctx, exchangeID, userID)
	if err != nil {
		return fmt.Errorf("decline exchange participation: %w", err)
	}

	// Отмена уже зафиксирована в БД. Ошибка дополнительного поиска не должна
	// превращать успешный decline в HTTP 500: клиент иначе повторит запрос к уже
	// закрытому обмену. Поэтому поиск выполняется best effort и только логируется.
	s.recoverExchanges(ctx, recoveryNodes)

	return nil
}

func (s *Service) recoverExchanges(ctx context.Context, nodes []exchangemodel.Node) {
	if len(nodes) < 2 {
		return
	}

	// Узлы освобождённых объявлений приходят из БД в порядке UUID, а не в порядке
	// передачи, поэтому направленную подпись отменённого цикла из них не строим.
	// Сохранённая UNIQUE-подпись отклонит точный повтор, после чего цикл попадёт
	// в этот локальный набор и DFS продолжит искать альтернативу.
	excludedCycles := make(map[string]struct{})

	for _, start := range nodes {
		for {
			cycle, err := s.findCycle(ctx, start, excludedCycles)
			if err != nil {
				s.logger.Printf("recover exchange search for item %s: %v", start.ItemID, err)
				break
			}
			if cycle == nil {
				break
			}

			signature := cycleKey(cycle)
			if _, err := s.SaveCycle(ctx, cycle); errors.Is(err, ErrDuplicateExchange) {
				excludedCycles[signature] = struct{}{}
				continue
			} else if err != nil {
				s.logger.Printf("save recovered exchange for item %s: %v", start.ItemID, err)
				break
			}

			// Один и тот же цикл можно обойти с каждого объявления. Каноническая
			// подпись не даёт сохранить его несколько раз за один recovery.
			excludedCycles[signature] = struct{}{}
			break
		}
	}
}

func cycleKey(cycle []exchangemodel.Node) string {
	transfers := make([]string, len(cycle))
	for index, node := range cycle {
		next := cycle[(index+1)%len(cycle)]
		transfers[index] = node.ItemID.String() + ">" + next.ItemID.String()
	}
	sort.Strings(transfers)
	return strings.Join(transfers, "|")
}

func (s *Service) CompleteParticipation(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) error {
	if err := s.repository.CompleteParticipation(ctx, exchangeID, userID); err != nil {
		return fmt.Errorf("complete exchange participation: %w", err)
	}

	return nil
}

// PostMessage добавляет сообщение участника в тред обмена.
func (s *Service) PostMessage(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
	body string,
) (exchangemodel.Message, error) {
	body = strings.TrimSpace(body)
	if length := utf8.RuneCountInString(body); length == 0 || length > maxMessageLength {
		return exchangemodel.Message{}, fmt.Errorf(
			"%w: message body must be between 1 and %d characters",
			ErrValidation,
			maxMessageLength,
		)
	}
	// Postgres не принимает NUL в text, поэтому без этой проверки такое тело доходит
	// до базы и возвращается пользователю как 500 вместо 400.
	if strings.ContainsRune(body, 0) {
		return exchangemodel.Message{}, fmt.Errorf(
			"%w: message body must not contain NUL characters",
			ErrValidation,
		)
	}

	status, isParticipant, err := s.repository.ExchangeAccess(ctx, exchangeID, userID)
	if err != nil {
		return exchangemodel.Message{}, fmt.Errorf("check exchange access: %w", err)
	}
	if !isParticipant {
		return exchangemodel.Message{}, ErrForbidden
	}
	// Тред закрытого обмена остаётся историей сделки и дописывать её уже нельзя.
	if status != exchangerepository.StatusProposed && status != exchangerepository.StatusConfirmed {
		return exchangemodel.Message{}, ErrConflict
	}

	message, err := s.repository.CreateMessage(ctx, exchangeID, userID, body)
	if err != nil {
		return exchangemodel.Message{}, fmt.Errorf("post exchange message: %w", err)
	}

	return message, nil
}

// ListMessages отдаёт тред целиком: он короткий, а пагинация ради десятка строк
// заставила бы frontend склеивать страницы при каждом опросе. Побочных эффектов нет:
// отметку о прочтении ставит отдельный MarkThreadRead, иначе фоновый опрос гасил бы
// счётчик непрочитанного вслепую.
func (s *Service) ListMessages(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) ([]exchangemodel.Message, error) {
	_, isParticipant, err := s.repository.ExchangeAccess(ctx, exchangeID, userID)
	if err != nil {
		return nil, fmt.Errorf("check exchange access: %w", err)
	}
	if !isParticipant {
		return nil, ErrForbidden
	}

	messages, err := s.repository.ListMessages(ctx, exchangeID)
	if err != nil {
		return nil, fmt.Errorf("list exchange messages: %w", err)
	}

	return messages, nil
}

// MarkThreadRead двигает отметку участника до сообщения, которое клиент показал последним.
// Статус обмена не проверяется: закрытый тред читать можно, значит и дочитывать тоже.
//
// TODO: created_at ставится на INSERT, а видимой строка становится на COMMIT, поэтому
// событие из длинной транзакции (согласие -> подтверждение обмена) может появиться в ленте
// с меткой «в прошлом» относительно уже прочитанного и не попасть в счётчик. Лечится только
// монотонной последовательностью, выданной на коммите; для текущих объёмов треда избыточно.
func (s *Service) MarkThreadRead(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
	lastMessageID uuid.UUID,
) error {
	_, isParticipant, err := s.repository.ExchangeAccess(ctx, exchangeID, userID)
	if err != nil {
		return fmt.Errorf("check exchange access: %w", err)
	}
	if !isParticipant {
		return ErrForbidden
	}

	if err := s.repository.MarkMessagesRead(ctx, exchangeID, userID, lastMessageID); err != nil {
		return fmt.Errorf("mark exchange messages read: %w", err)
	}

	return nil
}

func validateCycle(cycle []exchangemodel.Node) error {
	if len(cycle) < 2 || len(cycle) > maxParticipants {
		return fmt.Errorf("%w: participant count must be between 2 and %d", ErrInvalidCycle, maxParticipants)
	}

	items := make(map[uuid.UUID]struct{}, len(cycle))
	owners := make(map[uuid.UUID]struct{}, len(cycle))

	for _, node := range cycle {
		if node.ItemID == uuid.Nil || node.OwnerID == uuid.Nil {
			return fmt.Errorf("%w: item and owner IDs must not be empty", ErrInvalidCycle)
		}

		if _, exists := items[node.ItemID]; exists {
			return fmt.Errorf("%w: item %s is repeated", ErrInvalidCycle, node.ItemID)
		}
		items[node.ItemID] = struct{}{}

		if _, exists := owners[node.OwnerID]; exists {
			return fmt.Errorf("%w: owner %s is repeated", ErrInvalidCycle, node.OwnerID)
		}
		owners[node.OwnerID] = struct{}{}
	}

	return nil
}
