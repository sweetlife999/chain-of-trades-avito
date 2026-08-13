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
	exchangesearch "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/search"
)

const (
	maxParticipants       = 5
	maxSearchResults      = 10
	maxRankingCandidates  = 100
	rankingPoolMultiplier = 10
	// Тред нужен, чтобы договориться о передаче вещей, а не для длинных писем.
	maxMessageLength = 2000
)

var (
	ErrInvalidCycle       = errors.New("invalid exchange cycle")
	ErrValidation         = errors.New("invalid exchange input")
	ErrForbidden          = exchangerepository.ErrNotParticipant
	ErrConflict           = exchangerepository.ErrConflict
	ErrNotFound           = exchangerepository.ErrNotFound
	ErrDuplicateExchange  = exchangerepository.ErrDuplicateExchange
	ErrStaleSearchResult  = exchangerepository.ErrStaleSearchResult
	ErrUnknownPickupPoint = exchangerepository.ErrUnknownPickupPoint
)

type Repository interface {
	FindNeighbors(context.Context, uuid.UUID) ([]exchangemodel.Node, error)
	HasUserBlockConflict(context.Context, uuid.UUID, []uuid.UUID) (bool, error)
	GetSearchUserStats(context.Context, []uuid.UUID) (map[uuid.UUID]exchangemodel.SearchUserStats, error)
	GetSearchItemFilters(context.Context, []uuid.UUID) (map[uuid.UUID]exchangemodel.SearchItemFilters, error)
	SaveExchange(context.Context, exchangemodel.Exchange) (uuid.UUID, error)
	ListByUser(context.Context, uuid.UUID) ([]exchangemodel.Details, error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (exchangemodel.Details, error)
	ConfirmParticipation(context.Context, uuid.UUID, uuid.UUID) error
	DeclineParticipation(context.Context, uuid.UUID, uuid.UUID) ([]exchangemodel.Node, string, error)
	CancelByAdmin(context.Context, uuid.UUID, uuid.UUID) ([]exchangemodel.Node, string, error)
	MarkDeliveredByAdmin(context.Context, uuid.UUID, uuid.UUID) error
	CompleteParticipation(context.Context, uuid.UUID, uuid.UUID) error
	RecordItemPickup(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	ExchangeAccess(context.Context, uuid.UUID, uuid.UUID) (string, bool, error)
	CreateMessage(context.Context, uuid.UUID, uuid.UUID, string) (exchangemodel.Message, error)
	ListMessages(context.Context, uuid.UUID) ([]exchangemodel.Message, error)
	MarkMessagesRead(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
}

type Service struct {
	repository  Repository
	searchQueue SearchQueue
	logger      errorLogger
}

type errorLogger interface {
	Printf(string, ...any)
}

type SearchQueue interface {
	Enqueue(exchangesearch.Job) (bool, error)
}

func New(repository Repository, searchQueues ...SearchQueue) *Service {
	var searchQueue SearchQueue
	if len(searchQueues) > 0 {
		searchQueue = searchQueues[0]
	}

	return newWithDependencies(repository, searchQueue, log.Default())
}

func newWithDependencies(repository Repository, searchQueue SearchQueue, logger errorLogger) *Service {
	return &Service{repository: repository, searchQueue: searchQueue, logger: logger}
}

// FindCycle is the backwards-compatible single-result search. New automatic
// search uses FindCycles and persists several alternatives.
func (s *Service) FindCycle(ctx context.Context, start exchangemodel.Node) ([]exchangemodel.Node, error) {
	cycles, err := s.findCycles(ctx, start, 1, nil)
	if err != nil || len(cycles) == 0 {
		return nil, err
	}

	return cycles[0], nil
}

// FindCycles enumerates up to limit unique cycles. Limit is deliberately
// bounded: this path is still synchronous until the worker task is introduced.
func (s *Service) FindCycles(
	ctx context.Context,
	start exchangemodel.Node,
	limit int,
) ([][]exchangemodel.Node, error) {
	if limit < 1 || limit > maxSearchResults {
		return nil, fmt.Errorf("%w: cycle limit must be between 1 and %d", ErrValidation, maxSearchResults)
	}

	return s.findCycles(ctx, start, limit, nil)
}

func (s *Service) findCycles(
	ctx context.Context,
	start exchangemodel.Node,
	limit int,
	excludedCompositions map[string]struct{},
) ([][]exchangemodel.Node, error) {
	path := []exchangemodel.Node{start}
	visitedItems := map[uuid.UUID]struct{}{start.ItemID: {}}
	visitedOwners := map[uuid.UUID]struct{}{start.OwnerID: {}}
	foundCompositions := make(map[string]struct{}, limit)
	cycles := make([][]exchangemodel.Node, 0, limit)

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
				if len(path) >= 2 && cycleLengthAllowed(path) {
					candidate := append([]exchangemodel.Node(nil), path...)
					composition := compositionKey(candidate)
					if _, excluded := excludedCompositions[composition]; excluded {
						continue
					}
					if _, found := foundCompositions[composition]; found {
						continue
					}

					foundCompositions[composition] = struct{}{}
					cycles = append(cycles, candidate)
					if len(cycles) == limit {
						return true, nil
					}
				}

				continue
			}

			if len(path) >= maxParticipants || len(path)+1 > normalizedMaxChainLength(next) {
				continue
			}
			if len(path)+1 > pathMaxChainLength(path) {
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

			stop, err := dfs(next)

			path = path[:len(path)-1]
			delete(visitedItems, next.ItemID)
			delete(visitedOwners, next.OwnerID)

			if err != nil {
				return false, err
			}

			if stop {
				return true, nil
			}
		}

		return false, nil
	}

	_, err := dfs(start)
	if err != nil {
		return nil, err
	}

	return cycles, nil
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

// FindAndSave keeps the old single-result contract for existing callers.
func (s *Service) FindAndSave(
	ctx context.Context,
	start exchangemodel.Node,
) (exchangemodel.SearchResult, error) {
	results, err := s.findAndSave(ctx, []exchangemodel.Node{start}, 1, nil)
	if err != nil {
		return exchangemodel.SearchResult{}, err
	}
	if len(results) == 0 {
		return exchangemodel.SearchResult{}, nil
	}

	return exchangemodel.SearchResult{ExchangeID: results[0], Found: true}, nil
}

// FindAndSaveAll is the automatic search entry point. It saves several
// alternatives but never more than maxSearchResults per trigger.
func (s *Service) FindAndSaveAll(
	ctx context.Context,
	start exchangemodel.Node,
) (exchangemodel.SearchResults, error) {
	ids, err := s.findAndSave(ctx, []exchangemodel.Node{start}, maxSearchResults, nil)
	if err != nil {
		return exchangemodel.SearchResults{ExchangeIDs: ids, Found: len(ids) > 0}, err
	}

	return exchangemodel.SearchResults{ExchangeIDs: ids, Found: len(ids) > 0}, nil
}

// ScheduleSearch ставит обычный поиск в фоновую очередь. Отсутствие очереди оставлено как
// синхронный fallback для изолированных сервисов и интеграционных тестов.
func (s *Service) ScheduleSearch(ctx context.Context, start exchangemodel.Node) error {
	if s.searchQueue == nil {
		_, err := s.FindAndSaveAll(ctx, start)
		return err
	}

	_, err := s.searchQueue.Enqueue(exchangesearch.NewItemJob(start))
	if err != nil {
		return fmt.Errorf("enqueue exchange search: %w", err)
	}

	return nil
}

// ProcessSearchJob реализует контракт фонового worker. Исключения передаются внутри job,
// поэтому recovery не может немедленно собрать только что отменённый состав.
func (s *Service) ProcessSearchJob(ctx context.Context, job exchangesearch.Job) error {
	excludedCompositions := make(map[string]struct{}, len(job.ExcludedCompositions))
	for _, composition := range job.ExcludedCompositions {
		excludedCompositions[composition] = struct{}{}
	}

	_, err := s.findAndSave(ctx, job.Nodes, maxSearchResults, excludedCompositions)
	return err
}

func (s *Service) findAndSave(
	ctx context.Context,
	starts []exchangemodel.Node,
	limit int,
	excludedCompositions map[string]struct{},
) ([]uuid.UUID, error) {
	if excludedCompositions == nil {
		excludedCompositions = make(map[string]struct{})
	}
	exchangeIDs := make([]uuid.UUID, 0, limit)
	candidateLimit := min(limit*rankingPoolMultiplier, maxRankingCandidates)

	for len(exchangeIDs) < limit {
		cycles, err := s.findCandidateCycles(ctx, starts, candidateLimit, excludedCompositions)
		if err != nil {
			return exchangeIDs, fmt.Errorf("search exchanges: %w", err)
		}
		if len(cycles) == 0 {
			break
		}

		userIDs := cycleOwnerIDs(cycles)
		stats, err := s.repository.GetSearchUserStats(ctx, userIDs)
		if err != nil {
			return exchangeIDs, fmt.Errorf("load exchange ranking stats: %w", err)
		}
		candidateCount := len(cycles)
		filters, err := s.repository.GetSearchItemFilters(ctx, cycleItemIDs(cycles))
		if err != nil {
			return exchangeIDs, fmt.Errorf("load exchange search filters: %w", err)
		}
		cycles = filterCycles(cycles, stats, filters)

		for _, cycle := range exchangesearch.RankCyclesWithFilters(cycles, stats, filters) {
			exchangeID, err := s.SaveCycle(ctx, cycle)
			if errors.Is(err, ErrDuplicateExchange) || errors.Is(err, ErrStaleSearchResult) {
				continue
			}
			if err != nil {
				return exchangeIDs, fmt.Errorf("persist found exchange: %w", err)
			}

			exchangeIDs = append(exchangeIDs, exchangeID)
			if len(exchangeIDs) == limit {
				return exchangeIDs, nil
			}
		}

		// A partially filled pool means every start node was exhausted. A full
		// pool may have more candidates, so another batch is useful when some
		// ranked candidates became duplicate or stale before persistence.
		if candidateCount < candidateLimit {
			break
		}
	}

	return exchangeIDs, nil
}

func (s *Service) findCandidateCycles(
	ctx context.Context,
	starts []exchangemodel.Node,
	limit int,
	excludedCompositions map[string]struct{},
) ([][]exchangemodel.Node, error) {
	cycles := make([][]exchangemodel.Node, 0, limit)
	for _, start := range starts {
		if len(cycles) == limit {
			break
		}

		found, err := s.findCycles(ctx, start, limit-len(cycles), excludedCompositions)
		if err != nil {
			return nil, err
		}
		for _, cycle := range found {
			excludedCompositions[compositionKey(cycle)] = struct{}{}
			cycles = append(cycles, cycle)
		}
	}

	return cycles, nil
}

func normalizedMaxChainLength(node exchangemodel.Node) int {
	if node.MaxChainLength < 2 || node.MaxChainLength > maxParticipants {
		return maxParticipants
	}
	return int(node.MaxChainLength)
}

func pathMaxChainLength(path []exchangemodel.Node) int {
	maximum := maxParticipants
	for _, node := range path {
		maximum = min(maximum, normalizedMaxChainLength(node))
	}
	return maximum
}

func cycleLengthAllowed(cycle []exchangemodel.Node) bool {
	return len(cycle) <= pathMaxChainLength(cycle)
}

func cycleItemIDs(cycles [][]exchangemodel.Node) []uuid.UUID {
	unique := make(map[uuid.UUID]struct{})
	for _, cycle := range cycles {
		for _, node := range cycle {
			unique[node.ItemID] = struct{}{}
		}
	}
	result := make([]uuid.UUID, 0, len(unique))
	for id := range unique {
		result = append(result, id)
	}
	return result
}

func filterCycles(
	cycles [][]exchangemodel.Node,
	stats map[uuid.UUID]exchangemodel.SearchUserStats,
	filters map[uuid.UUID]exchangemodel.SearchItemFilters,
) [][]exchangemodel.Node {
	result := make([][]exchangemodel.Node, 0, len(cycles))
	for _, cycle := range cycles {
		if cycleMatchesFilters(cycle, stats, filters) {
			result = append(result, cycle)
		}
	}
	return result
}

func cycleMatchesFilters(
	cycle []exchangemodel.Node,
	stats map[uuid.UUID]exchangemodel.SearchUserStats,
	filters map[uuid.UUID]exchangemodel.SearchItemFilters,
) bool {
	for _, requester := range cycle {
		filter, found := filters[requester.ItemID]
		if !found {
			filter.MaxChainLength = maxParticipants
		}
		if filter.MaxChainLength >= 2 && len(cycle) > int(filter.MaxChainLength) {
			return false
		}
		for _, participant := range cycle {
			if participant.OwnerID == requester.OwnerID {
				continue
			}
			rating := 3.0
			if participantStats, exists := stats[participant.OwnerID]; exists {
				rating = participantStats.Rating
			}
			if rating < filter.MinParticipantRating {
				return false
			}
		}
	}
	return true
}

func cycleOwnerIDs(cycles [][]exchangemodel.Node) []uuid.UUID {
	unique := make(map[uuid.UUID]struct{})
	for _, cycle := range cycles {
		for _, node := range cycle {
			unique[node.OwnerID] = struct{}{}
		}
	}

	userIDs := make([]uuid.UUID, 0, len(unique))
	for userID := range unique {
		userIDs = append(userIDs, userID)
	}
	sort.Slice(userIDs, func(first, second int) bool {
		return userIDs[first].String() < userIDs[second].String()
	})

	return userIDs
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
	recoveryNodes, cancelledComposition, err := s.repository.DeclineParticipation(ctx, exchangeID, userID)
	if err != nil {
		return fmt.Errorf("decline exchange participation: %w", err)
	}

	// Отмена уже зафиксирована в БД. Ошибка дополнительного поиска не должна
	// превращать успешный decline в HTTP 500: клиент иначе повторит запрос к уже
	// закрытому обмену. Поэтому поиск выполняется best effort и только логируется.
	s.recoverExchanges(ctx, recoveryNodes, cancelledComposition)

	return nil
}

// CancelByAdmin принудительно закрывает активный обмен. В отличие от отказа
// участника, административная отмена не помечает никого виновным и не меняет
// пользовательскую статистику. Освободившиеся объявления сразу возвращаются в поиск.
func (s *Service) CancelByAdmin(ctx context.Context, exchangeID, adminID uuid.UUID) error {
	recoveryNodes, cancelledComposition, err := s.repository.CancelByAdmin(ctx, exchangeID, adminID)
	if err != nil {
		return fmt.Errorf("cancel exchange by admin: %w", err)
	}

	// Сама отмена уже зафиксирована транзакцией. Повторный поиск — best effort:
	// его сбой не должен превращать успешный административный запрос в HTTP 500.
	s.recoverExchanges(ctx, recoveryNodes, cancelledComposition)

	return nil
}

// MarkDeliveredByAdmin открывает участникам этап подтверждения получения.
func (s *Service) MarkDeliveredByAdmin(ctx context.Context, exchangeID, adminID uuid.UUID) error {
	if err := s.repository.MarkDeliveredByAdmin(ctx, exchangeID, adminID); err != nil {
		return fmt.Errorf("mark exchange delivered by admin: %w", err)
	}

	return nil
}

func (s *Service) recoverExchanges(
	ctx context.Context,
	nodes []exchangemodel.Node,
	cancelledComposition string,
) {
	if len(nodes) < 2 {
		return
	}

	job := exchangesearch.NewRecoveryJob(nodes, cancelledComposition)
	if s.searchQueue != nil {
		if _, err := s.searchQueue.Enqueue(job); err != nil {
			s.logger.Printf("enqueue exchange recovery after cancellation: %v", err)
		}
		return
	}

	if err := s.ProcessSearchJob(ctx, job); err != nil {
		s.logger.Printf("recover exchanges after cancellation: %v", err)
	}
}

func compositionKey(cycle []exchangemodel.Node) string {
	items := make([]string, len(cycle))
	for index, node := range cycle {
		items[index] = node.ItemID.String()
	}
	sort.Strings(items)
	return strings.Join(items, "|")
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

// RecordItemPickup запоминает пункт, в который владелец отнёс вещь. Живёт в этом сервисе,
// а не в item: транзакция трогает вещь, обмен и его тред разом, и переход в доставку обязан
// быть в ней же. Вещь вне обмена сдаётся тем же способом — обмена для этого не требуется.
func (s *Service) RecordItemPickup(
	ctx context.Context,
	itemID uuid.UUID,
	ownerID uuid.UUID,
	pickupPointID uuid.UUID,
) error {
	if err := s.repository.RecordItemPickup(ctx, itemID, ownerID, pickupPointID); err != nil {
		return fmt.Errorf("record item pickup: %w", err)
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
	// Тред закрытого обмена остаётся историей сделки и дописывать её уже нельзя. Пока
	// обмен едет, договариваться нужно как раз сильнее всего, поэтому открыты все
	// незакрытые статусы, включая доставку.
	switch status {
	case exchangerepository.StatusProposed, exchangerepository.StatusConfirmed,
		exchangerepository.StatusDelivering, exchangerepository.StatusDelivered:
	default:
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
