package search

import (
	"context"
	"log"
	"time"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

const (
	defaultSearchTimeout = 30 * time.Second
	defaultMaxAttempts   = 3
	defaultRetryDelay    = 250 * time.Millisecond
)

type Finder interface {
	FindAndSaveAll(context.Context, exchangemodel.Node) (exchangemodel.SearchResults, error)
}

type workerLogger interface {
	Printf(string, ...any)
}

type Worker struct {
	queue         *Queue
	finder        Finder
	logger        workerLogger
	searchTimeout time.Duration
	maxAttempts   int
	retryDelay    time.Duration
}

func NewWorker(queue *Queue, finder Finder) *Worker {
	return newWorker(
		queue,
		finder,
		log.Default(),
		defaultSearchTimeout,
		defaultMaxAttempts,
		defaultRetryDelay,
	)
}

func newWorker(
	queue *Queue,
	finder Finder,
	logger workerLogger,
	searchTimeout time.Duration,
	maxAttempts int,
	retryDelay time.Duration,
) *Worker {
	return &Worker{
		queue:         queue,
		finder:        finder,
		logger:        logger,
		searchTimeout: searchTimeout,
		maxAttempts:   maxAttempts,
		retryDelay:    retryDelay,
	}
}

// Run обрабатывает задачи последовательно, чтобы несколько тяжёлых DFS не создавали
// неконтролируемую нагрузку на PostgreSQL. Закрытая очередь дочитывается полностью;
// отменённый context прерывает worker немедленно и используется для аварийной остановки.
func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, open := <-w.queue.Jobs():
			if !open {
				return
			}

			w.process(ctx, job)
		}
	}
}

func (w *Worker) process(ctx context.Context, job Job) {
	defer w.queue.Complete(job.Node.ItemID)

	var lastErr error
	for attempt := 1; attempt <= w.maxAttempts; attempt++ {
		searchCtx, cancel := context.WithTimeout(ctx, w.searchTimeout)
		_, lastErr = w.finder.FindAndSaveAll(searchCtx, job.Node)
		cancel()
		if lastErr == nil {
			return
		}
		if ctx.Err() != nil {
			return
		}
		if attempt == w.maxAttempts {
			break
		}

		timer := time.NewTimer(w.retryDelay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-timer.C:
		}
	}

	w.logger.Printf(
		"asynchronous exchange search for item %s failed after %d attempts: %v",
		job.Node.ItemID,
		w.maxAttempts,
		lastErr,
	)
}
