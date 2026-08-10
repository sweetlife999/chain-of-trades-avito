package search

import (
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

var (
	ErrInvalidCapacity = errors.New("search queue capacity must be positive")
	ErrInvalidJob      = errors.New("search job must contain at least one valid node")
	ErrQueueFull       = errors.New("search queue is full")
	ErrQueueClosed     = errors.New("search queue is closed")
)

type Job struct {
	Nodes                []exchangemodel.Node
	ExcludedCompositions []string
	key                  string
}

func NewItemJob(node exchangemodel.Node) Job {
	return Job{
		Nodes: []exchangemodel.Node{node},
		key:   "item:" + node.ItemID.String(),
	}
}

func NewRecoveryJob(nodes []exchangemodel.Node, excludedComposition string) Job {
	itemIDs := make([]string, len(nodes))
	for index, node := range nodes {
		itemIDs[index] = node.ItemID.String()
	}
	sort.Strings(itemIDs)

	excluded := make([]string, 0, 1)
	if excludedComposition != "" {
		excluded = append(excluded, excludedComposition)
	}

	return Job{
		Nodes:                append([]exchangemodel.Node(nil), nodes...),
		ExcludedCompositions: excluded,
		key:                  "recovery:" + excludedComposition + ":" + strings.Join(itemIDs, ","),
	}
}

// Queue хранит только идентификаторы стартового объявления и владельца. Сам граф worker
// читает из PostgreSQL в момент выполнения, поэтому задача не содержит устаревающий снимок.
// Одно объявление одновременно может иметь только одну ожидающую или выполняемую задачу.
type Queue struct {
	mu       sync.Mutex
	jobs     chan Job
	inFlight map[string]struct{}
	closed   bool
}

func NewQueue(capacity int) (*Queue, error) {
	if capacity < 1 {
		return nil, ErrInvalidCapacity
	}

	return &Queue{
		jobs:     make(chan Job, capacity),
		inFlight: make(map[string]struct{}, capacity),
	}, nil
}

// Enqueue не блокирует HTTP-запрос. Если буфер исчерпан, вызывающий код получает явную
// ошибку и может записать её в лог, не откатывая уже сохранённое объявление.
// false без ошибки означает, что эквивалентная задача уже ожидает или выполняется.
func (q *Queue) Enqueue(job Job) (bool, error) {
	if !validJob(job) {
		return false, ErrInvalidJob
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return false, ErrQueueClosed
	}
	if _, exists := q.inFlight[job.key]; exists {
		return false, nil
	}

	job = cloneJob(job)
	select {
	case q.jobs <- job:
		q.inFlight[job.key] = struct{}{}
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
func (q *Queue) Complete(job Job) {
	q.mu.Lock()
	delete(q.inFlight, job.key)
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

func validJob(job Job) bool {
	if job.key == "" || len(job.Nodes) == 0 {
		return false
	}
	for _, node := range job.Nodes {
		if node.ItemID == uuid.Nil || node.OwnerID == uuid.Nil {
			return false
		}
	}

	return true
}

func cloneJob(job Job) Job {
	job.Nodes = append([]exchangemodel.Node(nil), job.Nodes...)
	job.ExcludedCompositions = append([]string(nil), job.ExcludedCompositions...)
	return job
}
