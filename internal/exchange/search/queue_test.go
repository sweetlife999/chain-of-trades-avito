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
	itemJob := NewItemJob(node)

	added, err := queue.Enqueue(itemJob)
	if err != nil || !added {
		t.Fatalf("first Enqueue() = (%t, %v), want (true, nil)", added, err)
	}
	added, err = queue.Enqueue(itemJob)
	if err != nil || added {
		t.Fatalf("duplicate Enqueue() = (%t, %v), want (false, nil)", added, err)
	}

	job := <-queue.Jobs()
	if len(job.Nodes) != 1 || job.Nodes[0] != node {
		t.Fatalf("job nodes = %+v, want [%+v]", job.Nodes, node)
	}
	queue.Complete(job)

	added, err = queue.Enqueue(itemJob)
	if err != nil || !added {
		t.Fatalf("Enqueue() after Complete = (%t, %v), want (true, nil)", added, err)
	}
}

func TestQueueFullDoesNotPoisonDeduplication(t *testing.T) {
	t.Parallel()

	queue := newTestQueue(t, 1)
	first := testNode()
	second := testNode()

	if added, err := queue.Enqueue(NewItemJob(first)); err != nil || !added {
		t.Fatalf("first Enqueue() = (%t, %v)", added, err)
	}
	if added, err := queue.Enqueue(NewItemJob(second)); !errors.Is(err, ErrQueueFull) || added {
		t.Fatalf("full Enqueue() = (%t, %v), want (false, %v)", added, err, ErrQueueFull)
	}

	<-queue.Jobs()
	if added, err := queue.Enqueue(NewItemJob(second)); err != nil || !added {
		t.Fatalf("second Enqueue() after free slot = (%t, %v), want (true, nil)", added, err)
	}
}

func TestQueueCloseIsIdempotentAndRejectsNewJobs(t *testing.T) {
	t.Parallel()

	queue := newTestQueue(t, 1)
	queue.Close()
	queue.Close()

	if added, err := queue.Enqueue(NewItemJob(testNode())); !errors.Is(err, ErrQueueClosed) || added {
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
	job := NewItemJob(node)
	const attempts = 32

	var waitGroup sync.WaitGroup
	results := make(chan bool, attempts)
	errorsCh := make(chan error, attempts)
	for range attempts {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			added, err := queue.Enqueue(job)
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

func TestQueueRejectsInvalidJob(t *testing.T) {
	t.Parallel()

	queue := newTestQueue(t, 1)
	for _, job := range []Job{
		{},
		NewItemJob(exchangemodel.Node{ItemID: uuid.New()}),
		NewItemJob(exchangemodel.Node{OwnerID: uuid.New()}),
	} {
		if added, err := queue.Enqueue(job); !errors.Is(err, ErrInvalidJob) || added {
			t.Fatalf("Enqueue(%+v) = (%t, %v), want (false, %v)", job, added, err, ErrInvalidJob)
		}
	}
}

func TestQueueCopiesRecoveryJob(t *testing.T) {
	t.Parallel()

	queue := newTestQueue(t, 1)
	nodes := []exchangemodel.Node{testNode(), testNode()}
	job := NewRecoveryJob(nodes, "cancelled-composition")
	if added, err := queue.Enqueue(job); err != nil || !added {
		t.Fatalf("Enqueue() = (%t, %v)", added, err)
	}

	nodes[0] = testNode()
	job.Nodes[1] = testNode()
	job.ExcludedCompositions[0] = "mutated"
	queued := <-queue.Jobs()
	if queued.Nodes[0] == nodes[0] || queued.Nodes[1] == job.Nodes[1] {
		t.Fatal("queued nodes changed through caller slice")
	}
	if queued.ExcludedCompositions[0] != "cancelled-composition" {
		t.Fatalf("excluded compositions = %v", queued.ExcludedCompositions)
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
