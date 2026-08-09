package search

import (
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

func TestNewQueueValidatesCapacity(t *testing.T) {
	t.Parallel()

	for _, capacity := range []int{-1, 0} {
		if _, err := NewQueue(capacity); !errors.Is(err, ErrInvalidCapacity) {
			t.Fatalf("NewQueue(%d) error = %v, want %v", capacity, err, ErrInvalidCapacity)
		}
	}
}

func TestQueueDeduplicatesItemUntilCompletion(t *testing.T) {
	t.Parallel()

	queue := newTestQueue(t, 2)
	node := testNode()

	added, err := queue.Enqueue(node)
	if err != nil || !added {
		t.Fatalf("first Enqueue() = (%t, %v), want (true, nil)", added, err)
	}
	added, err = queue.Enqueue(node)
	if err != nil || added {
		t.Fatalf("duplicate Enqueue() = (%t, %v), want (false, nil)", added, err)
	}

	job := <-queue.Jobs()
	if job.Node != node {
		t.Fatalf("job node = %+v, want %+v", job.Node, node)
	}
	queue.Complete(node.ItemID)

	added, err = queue.Enqueue(node)
	if err != nil || !added {
		t.Fatalf("Enqueue() after Complete = (%t, %v), want (true, nil)", added, err)
	}
}

func TestQueueFullDoesNotPoisonDeduplication(t *testing.T) {
	t.Parallel()

	queue := newTestQueue(t, 1)
	first := testNode()
	second := testNode()

	if added, err := queue.Enqueue(first); err != nil || !added {
		t.Fatalf("first Enqueue() = (%t, %v)", added, err)
	}
	if added, err := queue.Enqueue(second); !errors.Is(err, ErrQueueFull) || added {
		t.Fatalf("full Enqueue() = (%t, %v), want (false, %v)", added, err, ErrQueueFull)
	}

	<-queue.Jobs()
	if added, err := queue.Enqueue(second); err != nil || !added {
		t.Fatalf("second Enqueue() after free slot = (%t, %v), want (true, nil)", added, err)
	}
}

func TestQueueCloseIsIdempotentAndRejectsNewJobs(t *testing.T) {
	t.Parallel()

	queue := newTestQueue(t, 1)
	queue.Close()
	queue.Close()

	if added, err := queue.Enqueue(testNode()); !errors.Is(err, ErrQueueClosed) || added {
		t.Fatalf("Enqueue() after Close = (%t, %v), want (false, %v)", added, err, ErrQueueClosed)
	}
	if _, open := <-queue.Jobs(); open {
		t.Fatal("Jobs() channel is still open")
	}
}

func TestQueueConcurrentDuplicateEnqueue(t *testing.T) {
	t.Parallel()

	queue := newTestQueue(t, 32)
	node := testNode()
	const attempts = 32

	var waitGroup sync.WaitGroup
	results := make(chan bool, attempts)
	errorsCh := make(chan error, attempts)
	for range attempts {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			added, err := queue.Enqueue(node)
			results <- added
			errorsCh <- err
		}()
	}
	waitGroup.Wait()
	close(results)
	close(errorsCh)

	addedCount := 0
	for added := range results {
		if added {
			addedCount++
		}
	}
	for err := range errorsCh {
		if err != nil {
			t.Fatalf("concurrent Enqueue() error = %v", err)
		}
	}
	if addedCount != 1 {
		t.Fatalf("jobs added = %d, want 1", addedCount)
	}
}

func newTestQueue(t *testing.T, capacity int) *Queue {
	t.Helper()

	queue, err := NewQueue(capacity)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	t.Cleanup(queue.Close)
	return queue
}

func testNode() exchangemodel.Node {
	return exchangemodel.Node{ItemID: uuid.New(), OwnerID: uuid.New()}
}
