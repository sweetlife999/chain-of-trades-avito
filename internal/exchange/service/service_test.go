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

func TestSaveCycleBuildsParticipants(t *testing.T) {
	t.Parallel()

	nodes := makeNodes(3)
	exchangeID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	repository := &fakeRepository{savedExchangeID: exchangeID}

	actualID, err := New(repository).SaveCycle(context.Background(), nodes)
	if err != nil {
		t.Fatalf("SaveCycle() error = %v", err)
	}

	if actualID != exchangeID {
		t.Fatalf("SaveCycle() ID = %s, want %s", actualID, exchangeID)
	}

	want := exchangemodel.Exchange{Participants: []exchangemodel.Participant{
		{
			UserID:         nodes[0].OwnerID,
			GivesItemID:    nodes[0].ItemID,
			ReceivesItemID: nodes[1].ItemID,
			Position:       0,
		},
		{
			UserID:         nodes[1].OwnerID,
			GivesItemID:    nodes[1].ItemID,
			ReceivesItemID: nodes[2].ItemID,
			Position:       1,
		},
		{
			UserID:         nodes[2].OwnerID,
			GivesItemID:    nodes[2].ItemID,
			ReceivesItemID: nodes[0].ItemID,
			Position:       2,
		},
	}}

	if !reflect.DeepEqual(repository.savedExchange, want) {
		t.Fatalf("saved exchange = %+v, want %+v", repository.savedExchange, want)
	}
}

func TestSaveCycleRejectsInvalidCycle(t *testing.T) {
	t.Parallel()

	tests := map[string][]exchangemodel.Node{
		"empty":            nil,
		"one participant":  makeNodes(1),
		"six participants": makeNodes(6),
		"empty item ID": {
			{ItemID: uuid.Nil, OwnerID: uuid.New()},
			{ItemID: uuid.New(), OwnerID: uuid.New()},
		},
		"empty owner ID": {
			{ItemID: uuid.New(), OwnerID: uuid.Nil},
			{ItemID: uuid.New(), OwnerID: uuid.New()},
		},
		"repeated item": {
			{ItemID: testNode(1).ItemID, OwnerID: testNode(1).OwnerID},
			{ItemID: testNode(1).ItemID, OwnerID: testNode(2).OwnerID},
		},
		"repeated owner": {
			{ItemID: testNode(1).ItemID, OwnerID: testNode(1).OwnerID},
			{ItemID: testNode(2).ItemID, OwnerID: testNode(1).OwnerID},
		},
	}

	for name, cycle := range tests {
		cycle := cycle
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repository := &fakeRepository{}
			_, err := New(repository).SaveCycle(context.Background(), cycle)
			if !errors.Is(err, ErrInvalidCycle) {
				t.Fatalf("SaveCycle() error = %v, want %v", err, ErrInvalidCycle)
			}

			if repository.saveCalls != 0 {
				t.Fatalf("SaveExchange() calls = %d, want 0", repository.saveCalls)
			}
		})
	}
}

func TestSaveCycleRepositoryError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	repository := &fakeRepository{saveErr: databaseError}

	_, err := New(repository).SaveCycle(context.Background(), makeNodes(2))
	if !errors.Is(err, databaseError) {
		t.Fatalf("SaveCycle() error = %v, want wrapped %v", err, databaseError)
	}
}

func TestFindAndSave(t *testing.T) {
	t.Parallel()

	nodes := makeNodes(3)
	exchangeID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	repository := &fakeRepository{
		neighbors:       cycleGraph(nodes),
		savedExchangeID: exchangeID,
	}

	result, err := New(repository).FindAndSave(context.Background(), nodes[0])
	if err != nil {
		t.Fatalf("FindAndSave() error = %v", err)
	}

	if !result.Found {
		t.Fatal("FindAndSave() Found = false, want true")
	}

	if result.ExchangeID != exchangeID {
		t.Fatalf("FindAndSave() ID = %s, want %s", result.ExchangeID, exchangeID)
	}

	if repository.saveCalls != 1 {
		t.Fatalf("SaveExchange() calls = %d, want 1", repository.saveCalls)
	}
}

func TestFindAndSaveWithoutCycle(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	repository := &fakeRepository{neighbors: map[uuid.UUID][]exchangemodel.Node{
		start.ItemID: {},
	}}

	result, err := New(repository).FindAndSave(context.Background(), start)
	if err != nil {
		t.Fatalf("FindAndSave() error = %v", err)
	}

	if result.Found || result.ExchangeID != uuid.Nil {
		t.Fatalf("FindAndSave() = %+v, want empty result", result)
	}

	if repository.saveCalls != 0 {
		t.Fatalf("SaveExchange() calls = %d, want 0", repository.saveCalls)
	}
}

func TestFindAndSaveSearchError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("search failed")
	start := testNode(1)
	repository := &fakeRepository{errors: map[uuid.UUID]error{
		start.ItemID: databaseError,
	}}

	_, err := New(repository).FindAndSave(context.Background(), start)
	if !errors.Is(err, databaseError) {
		t.Fatalf("FindAndSave() error = %v, want wrapped %v", err, databaseError)
	}

	if repository.saveCalls != 0 {
		t.Fatalf("SaveExchange() calls = %d, want 0", repository.saveCalls)
	}
}

func TestFindAndSaveSaveError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("save failed")
	nodes := makeNodes(2)
	repository := &fakeRepository{
		neighbors: cycleGraph(nodes),
		saveErr:   databaseError,
	}

	_, err := New(repository).FindAndSave(context.Background(), nodes[0])
	if !errors.Is(err, databaseError) {
		t.Fatalf("FindAndSave() error = %v, want wrapped %v", err, databaseError)
	}
}

func TestListForUser(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	want := []exchangemodel.Details{{ID: uuid.New(), Status: "proposed"}}
	repository := &fakeRepository{listedExchanges: want}

	actual, err := New(repository).ListForUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}

	if repository.listedUserID != userID {
		t.Fatalf("ListByUser() user ID = %s, want %s", repository.listedUserID, userID)
	}

	if !reflect.DeepEqual(actual, want) {
		t.Fatalf("ListForUser() = %+v, want %+v", actual, want)
	}
}

func TestListForUserReturnsEmptySlice(t *testing.T) {
	t.Parallel()

	exchanges, err := New(&fakeRepository{}).ListForUser(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}

	if exchanges == nil {
		t.Fatal("ListForUser() returned nil, want []")
	}
}

func TestListForUserRepositoryError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	repository := &fakeRepository{listErr: databaseError}

	_, err := New(repository).ListForUser(context.Background(), uuid.New())
	if !errors.Is(err, databaseError) {
		t.Fatalf("ListForUser() error = %v, want wrapped %v", err, databaseError)
	}
}

func TestGetForUserAllowsParticipant(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	exchangeID := uuid.New()
	want := exchangemodel.Details{
		ID: exchangeID,
		Participants: []exchangemodel.DetailsParticipant{{
			User: exchangemodel.ParticipantUser{ID: userID},
		}},
	}
	repository := &fakeRepository{exchangeDetails: want}

	actual, err := New(repository).GetForUser(context.Background(), exchangeID, userID)
	if err != nil {
		t.Fatalf("GetForUser() error = %v", err)
	}

	if repository.requestedExchangeID != exchangeID {
		t.Fatalf("GetByID() exchange ID = %s, want %s", repository.requestedExchangeID, exchangeID)
	}

	if !reflect.DeepEqual(actual, want) {
		t.Fatalf("GetForUser() = %+v, want %+v", actual, want)
	}
}

func TestGetForUserRejectsNonParticipant(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{exchangeDetails: exchangemodel.Details{
		ID: uuid.New(),
		Participants: []exchangemodel.DetailsParticipant{{
			User: exchangemodel.ParticipantUser{ID: uuid.New()},
		}},
	}}

	_, err := New(repository).GetForUser(context.Background(), repository.exchangeDetails.ID, uuid.New())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("GetForUser() error = %v, want %v", err, ErrForbidden)
	}
}

func TestGetForUserRepositoryError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	repository := &fakeRepository{getErr: databaseError}

	_, err := New(repository).GetForUser(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, databaseError) {
		t.Fatalf("GetForUser() error = %v, want wrapped %v", err, databaseError)
	}
}

func TestConfirmParticipation(t *testing.T) {
	t.Parallel()

	exchangeID := uuid.New()
	userID := uuid.New()
	repository := &fakeRepository{}

	if err := New(repository).ConfirmParticipation(context.Background(), exchangeID, userID); err != nil {
		t.Fatalf("ConfirmParticipation() error = %v", err)
	}
	if repository.confirmedExchangeID != exchangeID || repository.confirmedUserID != userID {
		t.Fatalf(
			"ConfirmParticipation() repository args = (%s, %s), want (%s, %s)",
			repository.confirmedExchangeID,
			repository.confirmedUserID,
			exchangeID,
			userID,
		)
	}
}

func TestDeclineParticipation(t *testing.T) {
	t.Parallel()

	exchangeID := uuid.New()
	userID := uuid.New()
	repository := &fakeRepository{}

	if err := New(repository).DeclineParticipation(context.Background(), exchangeID, userID); err != nil {
		t.Fatalf("DeclineParticipation() error = %v", err)
	}
	if repository.declinedExchangeID != exchangeID || repository.declinedUserID != userID {
		t.Fatalf(
			"DeclineParticipation() repository args = (%s, %s), want (%s, %s)",
			repository.declinedExchangeID,
			repository.declinedUserID,
			exchangeID,
			userID,
		)
	}
}

func TestCompleteParticipation(t *testing.T) {
	t.Parallel()

	exchangeID := uuid.New()
	userID := uuid.New()
	repository := &fakeRepository{}

	if err := New(repository).CompleteParticipation(context.Background(), exchangeID, userID); err != nil {
		t.Fatalf("CompleteParticipation() error = %v", err)
	}
	if repository.completedExchangeID != exchangeID || repository.completedUserID != userID {
		t.Fatalf(
			"CompleteParticipation() repository args = (%s, %s), want (%s, %s)",
			repository.completedExchangeID,
			repository.completedUserID,
			exchangeID,
			userID,
		)
	}
}

func TestParticipationDecisionWrapsRepositoryError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	repository := &fakeRepository{confirmErr: databaseError, declineErr: databaseError, completeErr: databaseError}
	service := New(repository)

	if err := service.ConfirmParticipation(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, databaseError) {
		t.Fatalf("ConfirmParticipation() error = %v, want wrapped %v", err, databaseError)
	}
	if err := service.DeclineParticipation(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, databaseError) {
		t.Fatalf("DeclineParticipation() error = %v, want wrapped %v", err, databaseError)
	}
	if err := service.CompleteParticipation(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, databaseError) {
		t.Fatalf("CompleteParticipation() error = %v, want wrapped %v", err, databaseError)
	}
}

type fakeRepository struct {
	neighbors           map[uuid.UUID][]exchangemodel.Node
	errors              map[uuid.UUID]error
	calls               int
	savedExchange       exchangemodel.Exchange
	savedExchangeID     uuid.UUID
	saveErr             error
	saveCalls           int
	listedExchanges     []exchangemodel.Details
	listedUserID        uuid.UUID
	listErr             error
	exchangeDetails     exchangemodel.Details
	requestedExchangeID uuid.UUID
	getErr              error
	confirmedExchangeID uuid.UUID
	confirmedUserID     uuid.UUID
	confirmErr          error
	declinedExchangeID  uuid.UUID
	declinedUserID      uuid.UUID
	declineErr          error
	completedExchangeID uuid.UUID
	completedUserID     uuid.UUID
	completeErr         error
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

func (f *fakeRepository) SaveExchange(
	_ context.Context,
	exchange exchangemodel.Exchange,
) (uuid.UUID, error) {
	f.saveCalls++
	f.savedExchange = exchange
	return f.savedExchangeID, f.saveErr
}

func (f *fakeRepository) ListByUser(
	_ context.Context,
	userID uuid.UUID,
) ([]exchangemodel.Details, error) {
	f.listedUserID = userID
	return f.listedExchanges, f.listErr
}

func (f *fakeRepository) GetByID(
	_ context.Context,
	exchangeID uuid.UUID,
) (exchangemodel.Details, error) {
	f.requestedExchangeID = exchangeID
	return f.exchangeDetails, f.getErr
}

func (f *fakeRepository) ConfirmParticipation(
	_ context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) error {
	f.confirmedExchangeID = exchangeID
	f.confirmedUserID = userID
	return f.confirmErr
}

func (f *fakeRepository) DeclineParticipation(
	_ context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) error {
	f.declinedExchangeID = exchangeID
	f.declinedUserID = userID
	return f.declineErr
}

func (f *fakeRepository) CompleteParticipation(
	_ context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) error {
	f.completedExchangeID = exchangeID
	f.completedUserID = userID
	return f.completeErr
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
