package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

func TestFindCycleSupportedLengths(t *testing.T) {
	t.Parallel()

	for _, participants := range []int{2, 3, 5} {
		participants := participants
		t.Run(cycleTestName(participants), func(t *testing.T) {
			t.Parallel()

			nodes := makeNodes(participants)
			repository := &fakeRepository{neighbors: cycleGraph(nodes)}

			cycle, err := New(repository).FindCycle(context.Background(), nodes[0])
			if err != nil {
				t.Fatalf("FindCycle() error = %v", err)
			}

			assertCycle(t, cycle, nodes)
		})
	}
}

func TestFindCycleRejectsSixParticipants(t *testing.T) {
	t.Parallel()

	nodes := makeNodes(6)
	repository := &fakeRepository{neighbors: cycleGraph(nodes)}

	cycle, err := New(repository).FindCycle(context.Background(), nodes[0])
	if err != nil {
		t.Fatalf("FindCycle() error = %v", err)
	}

	if cycle != nil {
		t.Fatalf("FindCycle() = %+v, want no cycle", cycle)
	}
}

func TestFindCycleBacktracksFromDeadEnd(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	deadEnd := testNode(2)
	secondBranch := testNode(3)
	last := testNode(4)

	repository := &fakeRepository{neighbors: map[uuid.UUID][]exchangemodel.Node{
		start.ItemID:        {deadEnd, secondBranch},
		deadEnd.ItemID:      {},
		secondBranch.ItemID: {last},
		last.ItemID:         {start},
	}}

	cycle, err := New(repository).FindCycle(context.Background(), start)
	if err != nil {
		t.Fatalf("FindCycle() error = %v", err)
	}

	assertCycle(t, cycle, []exchangemodel.Node{start, secondBranch, last})
}

func TestFindCycleReturnsFirstCycle(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	first := testNode(2)
	second := testNode(3)

	repository := &fakeRepository{neighbors: map[uuid.UUID][]exchangemodel.Node{
		start.ItemID:  {first, second},
		first.ItemID:  {start},
		second.ItemID: {start},
	}}

	cycle, err := New(repository).FindCycle(context.Background(), start)
	if err != nil {
		t.Fatalf("FindCycle() error = %v", err)
	}

	assertCycle(t, cycle, []exchangemodel.Node{start, first})
}

func TestFindCycleSkipsRepeatedOwner(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	middle := testNode(2)
	repeatedOwner := testNode(3)
	repeatedOwner.OwnerID = start.OwnerID

	repository := &fakeRepository{neighbors: map[uuid.UUID][]exchangemodel.Node{
		start.ItemID:         {middle},
		middle.ItemID:        {repeatedOwner},
		repeatedOwner.ItemID: {start},
	}}

	cycle, err := New(repository).FindCycle(context.Background(), start)
	if err != nil {
		t.Fatalf("FindCycle() error = %v", err)
	}

	if cycle != nil {
		t.Fatalf("FindCycle() = %+v, want no cycle", cycle)
	}
}

func TestFindCycleSkipsRepeatedItem(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	second := testNode(2)
	third := testNode(3)

	repository := &fakeRepository{neighbors: map[uuid.UUID][]exchangemodel.Node{
		start.ItemID:  {second},
		second.ItemID: {third},
		third.ItemID:  {second},
	}}

	cycle, err := New(repository).FindCycle(context.Background(), start)
	if err != nil {
		t.Fatalf("FindCycle() error = %v", err)
	}

	if cycle != nil {
		t.Fatalf("FindCycle() = %+v, want no cycle", cycle)
	}
}

func TestFindCycleNoCycle(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	deadEnd := testNode(2)
	repository := &fakeRepository{neighbors: map[uuid.UUID][]exchangemodel.Node{
		start.ItemID:   {deadEnd},
		deadEnd.ItemID: {},
	}}

	cycle, err := New(repository).FindCycle(context.Background(), start)
	if err != nil {
		t.Fatalf("FindCycle() error = %v", err)
	}

	if cycle != nil {
		t.Fatalf("FindCycle() = %+v, want no cycle", cycle)
	}
}

func TestFindCycleRepositoryError(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	second := testNode(2)
	databaseError := errors.New("database unavailable")

	repository := &fakeRepository{
		neighbors: map[uuid.UUID][]exchangemodel.Node{start.ItemID: {second}},
		errors:    map[uuid.UUID]error{second.ItemID: databaseError},
	}

	_, err := New(repository).FindCycle(context.Background(), start)
	if !errors.Is(err, databaseError) {
		t.Fatalf("FindCycle() error = %v, want wrapped %v", err, databaseError)
	}
}

func TestFindCycleCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repository := &fakeRepository{}
	_, err := New(repository).FindCycle(ctx, testNode(1))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("FindCycle() error = %v, want %v", err, context.Canceled)
	}

	if repository.calls != 0 {
		t.Fatalf("repository calls = %d, want 0", repository.calls)
	}
}

type fakeRepository struct {
	neighbors map[uuid.UUID][]exchangemodel.Node
	errors    map[uuid.UUID]error
	calls     int
}

func (f *fakeRepository) FindNeighbors(
	_ context.Context,
	itemID uuid.UUID,
) ([]exchangemodel.Node, error) {
	f.calls++

	if err := f.errors[itemID]; err != nil {
		return nil, err
	}

	return append([]exchangemodel.Node(nil), f.neighbors[itemID]...), nil
}

func cycleGraph(nodes []exchangemodel.Node) map[uuid.UUID][]exchangemodel.Node {
	graph := make(map[uuid.UUID][]exchangemodel.Node, len(nodes))
	for index, node := range nodes {
		graph[node.ItemID] = []exchangemodel.Node{nodes[(index+1)%len(nodes)]}
	}

	return graph
}

func makeNodes(count int) []exchangemodel.Node {
	nodes := make([]exchangemodel.Node, count)
	for index := range nodes {
		nodes[index] = testNode(byte(index + 1))
	}

	return nodes
}

func testNode(value byte) exchangemodel.Node {
	var itemID uuid.UUID
	itemID[len(itemID)-1] = value

	var ownerID uuid.UUID
	ownerID[len(ownerID)-1] = value

	return exchangemodel.Node{ItemID: itemID, OwnerID: ownerID}
}

func assertCycle(t *testing.T, actual, expected []exchangemodel.Node) {
	t.Helper()

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("cycle = %+v, want %+v", actual, expected)
	}
}

func cycleTestName(participants int) string {
	switch participants {
	case 2:
		return "two participants"
	case 3:
		return "three participants"
	case 5:
		return "five participants"
	default:
		return "unsupported length"
	}
}
