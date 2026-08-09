package search

import (
	"errors"
	"sync"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

var (
	ErrInvalidCapacity = errors.New("search queue capacity must be positive")
	ErrQueueFull       = errors.New("search queue is full")
	ErrQueueClosed     = errors.New("search queue is closed")
)

type Job struct {
	Node exchangemodel.Node
}

// Queue хранит только идентификаторы стартового объявления и владельца. Сам граф worker
// читает из PostgreSQL в момент выполнения, поэтому задача не содержит устаревающий снимок.
// Одно объявление одновременно может иметь только одну ожидающую или выполняемую задачу.
type Queue struct {
	mu       sync.Mutex
	jobs     chan Job
	inFlight map[uuid.UUID]struct{}
	closed   bool
}

func NewQueue(capacity int) (*Queue, error) {
	if capacity < 1 {
		return nil, ErrInvalidCapacity
	}

	return &Queue{
		jobs:     make(chan Job, capacity),
		inFlight: make(map[uuid.UUID]struct{}, capacity),
	}, nil
}

// Enqueue не блокирует HTTP-запрос. Если буфер исчерпан, вызывающий код получает явную
// ошибку и может записать её в лог, не откатывая уже сохранённое объявление.
// false без ошибки означает, что задача для этого объявления уже есть.
func (q *Queue) Enqueue(node exchangemodel.Node) (bool, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return false, ErrQueueClosed
	}
	if _, exists := q.inFlight[node.ItemID]; exists {
		return false, nil
	}

	select {
	case q.jobs <- Job{Node: node}:
		q.inFlight[node.ItemID] = struct{}{}
		return true, nil
	default:
		return false, ErrQueueFull
	}
}

func (q *Queue) Jobs() <-chan Job {
	return q.jobs
}

// Complete разрешает поставить объявление в очередь повторно после окончания поиска.
// Вызов безопасен и после Close: worker может завершать уже полученную задачу при остановке.
func (q *Queue) Complete(itemID uuid.UUID) {
	q.mu.Lock()
	delete(q.inFlight, itemID)
	q.mu.Unlock()
}

// Close прекращает приём новых задач и даёт worker дочитать уже поставленные. Повторный
// вызов безопасен.
func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return
	}

	q.closed = true
	close(q.jobs)
}
