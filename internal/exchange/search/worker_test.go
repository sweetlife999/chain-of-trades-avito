package search

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

func TestWorkerProcessesJobAndReleasesDeduplication(t *testing.T) {
	t.Parallel()

	queue := newTestQueue(t, 1)
	finder := newFakeFinder()
	worker := testWorker(queue, finder, log.Default())
	node := testNode()
	job := NewItemJob(node)
	if added, err := queue.Enqueue(job); err != nil || !added {
		t.Fatalf("Enqueue() = (%t, %v)", added, err)
	}

	runDone := make(chan struct{})
	go func() {
		worker.Run(context.Background())
		close(runDone)
	}()

	waitJob(t, finder.called, job)
	waitEnqueue(t, queue, job)
	queue.Close()
	waitJob(t, finder.called, job)
	waitSignal(t, runDone, "worker did not drain closed queue")
}

func TestWorkerRetriesAndEventuallySucceeds(t *testing.T) {
	t.Parallel()

	queue := newTestQueue(t, 1)
	finder := newFakeFinder(errors.New("temporary one"), errors.New("temporary two"), nil)
	worker := testWorker(queue, finder, log.Default())
	if added, err := queue.Enqueue(NewItemJob(testNode())); err != nil || !added {
		t.Fatalf("Enqueue() = (%t, %v)", added, err)
	}
	queue.Close()

	worker.Run(context.Background())
	if calls := finder.callCount(); calls != 3 {
		t.Fatalf("FindAndSaveAll() calls = %d, want 3", calls)
	}
}

func TestWorkerLogsFinalErrorAndContinues(t *testing.T) {
	t.Parallel()

	queue := newTestQueue(t, 2)
	finder := newFakeFinder(
		errors.New("database unavailable"),
		errors.New("database unavailable"),
		errors.New("database unavailable"),
		nil,
	)
	var logs bytes.Buffer
	worker := testWorker(queue, finder, log.New(&logs, "", 0))
	first, second := testNode(), testNode()
	for _, node := range []exchangemodel.Node{first, second} {
		if added, err := queue.Enqueue(NewItemJob(node)); err != nil || !added {
			t.Fatalf("Enqueue() = (%t, %v)", added, err)
		}
	}
	queue.Close()

	worker.Run(context.Background())
	if calls := finder.callCount(); calls != 4 {
		t.Fatalf("FindAndSaveAll() calls = %d, want 4", calls)
	}
	if !strings.Contains(logs.String(), NewItemJob(first).key) ||
		!strings.Contains(logs.String(), "failed after 3 attempts") {
		t.Fatalf("worker log = %q", logs.String())
	}
}

func TestWorkerCancellationInterruptsRetryDelay(t *testing.T) {
	t.Parallel()

	queue := newTestQueue(t, 1)
	finder := newFakeFinder(errors.New("temporary"))
	worker := newWorker(queue, finder, log.Default(), time.Second, 3, time.Hour)
	if added, err := queue.Enqueue(NewItemJob(testNode())); err != nil || !added {
		t.Fatalf("Enqueue() = (%t, %v)", added, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(runDone)
	}()
	<-finder.called
	cancel()

	waitSignal(t, runDone, "worker ignored cancellation during retry delay")
}

func TestWorkerAppliesSearchTimeout(t *testing.T) {
	t.Parallel()

	queue := newTestQueue(t, 1)
	finder := &blockingFinder{contexts: make(chan context.Context, 1)}
	worker := newWorker(queue, finder, log.Default(), 10*time.Millisecond, 1, time.Millisecond)
	if added, err := queue.Enqueue(NewItemJob(testNode())); err != nil || !added {
		t.Fatalf("Enqueue() = (%t, %v)", added, err)
	}
	queue.Close()

	worker.Run(context.Background())
	searchCtx := <-finder.contexts
	if !errors.Is(searchCtx.Err(), context.DeadlineExceeded) {
		t.Fatalf("search context error = %v, want %v", searchCtx.Err(), context.DeadlineExceeded)
	}
}

type fakeFinder struct {
	mu     sync.Mutex
	errors []error
	calls  int
	called chan Job
}

func newFakeFinder(results ...error) *fakeFinder {
	return &fakeFinder{
		errors: results,
		called: make(chan Job, 16),
	}
}

func (f *fakeFinder) ProcessSearchJob(
	_ context.Context,
	job Job,
) error {
	f.mu.Lock()
	index := f.calls
	f.calls++
	var err error
	if index < len(f.errors) {
		err = f.errors[index]
	}
	f.mu.Unlock()

	f.called <- job
	return err
}

func (f *fakeFinder) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

type blockingFinder struct {
	contexts chan context.Context
}

func (f *blockingFinder) ProcessSearchJob(
	ctx context.Context,
	_ Job,
) error {
	f.contexts <- ctx
	<-ctx.Done()
	return ctx.Err()
}

func testWorker(queue *Queue, finder Finder, logger workerLogger) *Worker {
	return newWorker(queue, finder, logger, time.Second, 3, time.Millisecond)
}

func waitJob(t *testing.T, jobs <-chan Job, want Job) {
	t.Helper()

	select {
	case got := <-jobs:
		if got.key != want.key {
			t.Fatalf("processed job = %+v, want %+v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not process job")
	}
}

func waitSignal(t *testing.T, signal <-chan struct{}, failure string) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal(failure)
	}
}

func waitEnqueue(t *testing.T, queue *Queue, job Job) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		added, err := queue.Enqueue(job)
		if err != nil {
			t.Fatalf("Enqueue() after processing error = %v", err)
		}
		if added {
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatal("processed item remained marked as in flight")
}
