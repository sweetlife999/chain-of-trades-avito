package service

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	exchangesearch "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/search"
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

func TestFindCyclesReturnsSeveralAlternatives(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	alternatives := []exchangemodel.Node{testNode(2), testNode(3), testNode(4)}
	neighbors := map[uuid.UUID][]exchangemodel.Node{start.ItemID: alternatives}
	for _, alternative := range alternatives {
		neighbors[alternative.ItemID] = []exchangemodel.Node{start}
	}

	cycles, err := New(&fakeRepository{neighbors: neighbors}).FindCycles(
		context.Background(),
		start,
		maxSearchResults,
	)
	if err != nil {
		t.Fatalf("FindCycles() error = %v", err)
	}
	if len(cycles) != len(alternatives) {
		t.Fatalf("FindCycles() count = %d, want %d", len(cycles), len(alternatives))
	}
	for index, cycle := range cycles {
		assertCycle(t, cycle, []exchangemodel.Node{start, alternatives[index]})
	}
}

func TestFindCyclesDeduplicatesDifferentDirectionsOfSameComposition(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	second := testNode(2)
	third := testNode(3)
	repository := &fakeRepository{neighbors: map[uuid.UUID][]exchangemodel.Node{
		start.ItemID:  {second, third},
		second.ItemID: {third, start},
		third.ItemID:  {start, second},
	}}

	cycles, err := New(repository).FindCycles(context.Background(), start, maxSearchResults)
	if err != nil {
		t.Fatalf("FindCycles() error = %v", err)
	}
	wantComposition := compositionKey([]exchangemodel.Node{start, second, third})
	matches := 0
	for _, cycle := range cycles {
		if compositionKey(cycle) == wantComposition {
			matches++
		}
	}
	if matches != 1 {
		t.Fatalf("same three-item composition returned %d times, want 1", matches)
	}
}

func TestFindCyclesHonorsLimit(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	alternatives := []exchangemodel.Node{testNode(2), testNode(3), testNode(4)}
	neighbors := map[uuid.UUID][]exchangemodel.Node{start.ItemID: alternatives}
	for _, alternative := range alternatives {
		neighbors[alternative.ItemID] = []exchangemodel.Node{start}
	}

	cycles, err := New(&fakeRepository{neighbors: neighbors}).FindCycles(context.Background(), start, 2)
	if err != nil {
		t.Fatalf("FindCycles() error = %v", err)
	}
	if len(cycles) != 2 {
		t.Fatalf("FindCycles() count = %d, want 2", len(cycles))
	}
}

func TestFindCyclesRejectsInvalidLimit(t *testing.T) {
	t.Parallel()

	service := New(&fakeRepository{})
	for _, limit := range []int{0, maxSearchResults + 1} {
		if _, err := service.FindCycles(context.Background(), testNode(1), limit); !errors.Is(err, ErrValidation) {
			t.Fatalf("FindCycles(limit=%d) error = %v, want %v", limit, err, ErrValidation)
		}
	}
}

func TestFindCycleSkipsBlockedUserAndUsesAnotherBranch(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	blocked := testNode(2)
	allowed := testNode(3)
	repository := &fakeRepository{
		neighbors: map[uuid.UUID][]exchangemodel.Node{
			start.ItemID:   {blocked, allowed},
			blocked.ItemID: {start},
			allowed.ItemID: {start},
		},
		blockConflicts: map[uuid.UUID]bool{blocked.OwnerID: true},
	}

	cycle, err := New(repository).FindCycle(context.Background(), start)
	if err != nil {
		t.Fatalf("FindCycle() error = %v", err)
	}

	assertCycle(t, cycle, []exchangemodel.Node{start, allowed})
	if !reflect.DeepEqual(repository.blockChecks[blocked.OwnerID], []uuid.UUID{start.OwnerID}) {
		t.Fatalf(
			"blocked candidate was checked against %v, want start owner %v",
			repository.blockChecks[blocked.OwnerID],
			start.OwnerID,
		)
	}
}

func TestFindCycleChecksCandidateAgainstWholePath(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	middle := testNode(2)
	blockedLast := testNode(3)
	repository := &fakeRepository{
		neighbors: map[uuid.UUID][]exchangemodel.Node{
			start.ItemID:       {middle},
			middle.ItemID:      {blockedLast},
			blockedLast.ItemID: {start},
		},
		blockConflicts: map[uuid.UUID]bool{blockedLast.OwnerID: true},
	}

	cycle, err := New(repository).FindCycle(context.Background(), start)
	if err != nil {
		t.Fatalf("FindCycle() error = %v", err)
	}
	if cycle != nil {
		t.Fatalf("FindCycle() = %+v, want no cycle", cycle)
	}

	wantPath := []uuid.UUID{start.OwnerID, middle.OwnerID}
	if !reflect.DeepEqual(repository.blockChecks[blockedLast.OwnerID], wantPath) {
		t.Fatalf("block check path = %v, want %v", repository.blockChecks[blockedLast.OwnerID], wantPath)
	}
}

func TestFindCycleBlockCheckError(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	next := testNode(2)
	databaseError := errors.New("database unavailable")
	repository := &fakeRepository{
		neighbors:           map[uuid.UUID][]exchangemodel.Node{start.ItemID: {next}},
		blockConflictErrors: map[uuid.UUID]error{next.OwnerID: databaseError},
	}

	_, err := New(repository).FindCycle(context.Background(), start)
	if !errors.Is(err, databaseError) {
		t.Fatalf("FindCycle() error = %v, want wrapped %v", err, databaseError)
	}
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

func TestFindAndSaveSkipsDuplicateAndSavesAlternative(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	first := testNode(2)
	alternative := testNode(3)
	repository := &fakeRepository{
		neighbors: map[uuid.UUID][]exchangemodel.Node{
			start.ItemID:       {first, alternative},
			first.ItemID:       {start},
			alternative.ItemID: {start},
		},
		savedExchangeID: uuid.New(),
		saveErrors:      []error{ErrDuplicateExchange, nil},
	}

	result, err := New(repository).FindAndSave(context.Background(), start)
	if err != nil {
		t.Fatalf("FindAndSave() error = %v", err)
	}
	if !result.Found {
		t.Fatal("FindAndSave() Found = false, want alternative exchange")
	}
	if repository.saveCalls != 2 {
		t.Fatalf("SaveExchange() calls = %d, want 2", repository.saveCalls)
	}
	if len(repository.savedExchange.Participants) != 2 ||
		repository.savedExchange.Participants[1].GivesItemID != alternative.ItemID {
		t.Fatalf("saved exchange = %+v, want cycle through alternative item", repository.savedExchange)
	}
}

func TestFindAndSaveAllSavesSeveralAlternatives(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	alternatives := []exchangemodel.Node{testNode(2), testNode(3), testNode(4)}
	neighbors := map[uuid.UUID][]exchangemodel.Node{start.ItemID: alternatives}
	for _, alternative := range alternatives {
		neighbors[alternative.ItemID] = []exchangemodel.Node{start}
	}
	repository := &fakeRepository{neighbors: neighbors, savedExchangeID: uuid.New()}

	result, err := New(repository).FindAndSaveAll(context.Background(), start)
	if err != nil {
		t.Fatalf("FindAndSaveAll() error = %v", err)
	}
	if !result.Found || len(result.ExchangeIDs) != len(alternatives) {
		t.Fatalf("FindAndSaveAll() = %+v, want %d results", result, len(alternatives))
	}
	if len(repository.savedExchanges) != len(alternatives) {
		t.Fatalf("saved exchanges = %d, want %d", len(repository.savedExchanges), len(alternatives))
	}
}

func TestFindAndSaveAllRanksCandidatesBeforeSaving(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	lowRated := testNode(2)
	highRated := testNode(3)
	repository := &fakeRepository{
		neighbors: map[uuid.UUID][]exchangemodel.Node{
			start.ItemID:     {lowRated, highRated},
			lowRated.ItemID:  {start},
			highRated.ItemID: {start},
		},
		searchStats: map[uuid.UUID]exchangemodel.SearchUserStats{
			start.OwnerID:     {UserID: start.OwnerID, Rating: 3},
			lowRated.OwnerID:  {UserID: lowRated.OwnerID, Rating: 1},
			highRated.OwnerID: {UserID: highRated.OwnerID, Rating: 5},
		},
		savedExchangeID: uuid.New(),
	}

	result, err := New(repository).FindAndSaveAll(context.Background(), start)
	if err != nil {
		t.Fatalf("FindAndSaveAll() error = %v", err)
	}
	if len(result.ExchangeIDs) != 2 || len(repository.savedExchanges) != 2 {
		t.Fatalf("results = %d, saves = %d, want 2", len(result.ExchangeIDs), len(repository.savedExchanges))
	}
	if repository.savedExchanges[0].Participants[1].GivesItemID != highRated.ItemID {
		t.Fatalf("first saved exchange = %+v, want high-rated alternative", repository.savedExchanges[0])
	}
}

func TestFindCyclesRespectsMaximumLengthOfEveryItem(t *testing.T) {
	t.Parallel()
	start := testNode(1)
	second := testNode(2)
	third := testNode(3)
	second.MaxChainLength = 2
	repository := &fakeRepository{neighbors: map[uuid.UUID][]exchangemodel.Node{
		start.ItemID:  {second},
		second.ItemID: {third},
		third.ItemID:  {start},
	}}

	cycles, err := New(repository).FindCycles(context.Background(), start, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 0 {
		t.Fatalf("cycles = %v, want none because an item allows at most 2 participants", cycles)
	}
}

func TestCycleMatchesMinimumRatingOfEveryParticipant(t *testing.T) {
	t.Parallel()
	first := testNode(1)
	second := testNode(2)
	filters := map[uuid.UUID]exchangemodel.SearchItemFilters{
		first.ItemID:  {ItemID: first.ItemID, MaxChainLength: 5, MinParticipantRating: 4},
		second.ItemID: {ItemID: second.ItemID, MaxChainLength: 5},
	}
	stats := map[uuid.UUID]exchangemodel.SearchUserStats{
		first.OwnerID:  {UserID: first.OwnerID, Rating: 5},
		second.OwnerID: {UserID: second.OwnerID, Rating: 3.9},
	}

	if cycleMatchesFilters([]exchangemodel.Node{first, second}, stats, filters) {
		t.Fatal("cycle with participant below requested rating was accepted")
	}
	stats[second.OwnerID] = exchangemodel.SearchUserStats{UserID: second.OwnerID, Rating: 4}
	if !cycleMatchesFilters([]exchangemodel.Node{first, second}, stats, filters) {
		t.Fatal("cycle meeting the exact rating boundary was rejected")
	}
}

func TestFindAndSaveAllRanksBeyondResultLimit(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	alternatives := make([]exchangemodel.Node, maxSearchResults+2)
	neighbors := make(map[uuid.UUID][]exchangemodel.Node, len(alternatives)+1)
	stats := map[uuid.UUID]exchangemodel.SearchUserStats{
		start.OwnerID: {UserID: start.OwnerID, Rating: 3},
	}
	for index := range alternatives {
		alternatives[index] = testNode(byte(index + 2))
		neighbors[alternatives[index].ItemID] = []exchangemodel.Node{start}
		stats[alternatives[index].OwnerID] = exchangemodel.SearchUserStats{
			UserID: alternatives[index].OwnerID,
			Rating: 1,
		}
	}
	best := alternatives[len(alternatives)-1]
	stats[best.OwnerID] = exchangemodel.SearchUserStats{UserID: best.OwnerID, Rating: 5}
	neighbors[start.ItemID] = alternatives
	repository := &fakeRepository{
		neighbors:       neighbors,
		searchStats:     stats,
		savedExchangeID: uuid.New(),
	}

	result, err := New(repository).FindAndSaveAll(context.Background(), start)
	if err != nil {
		t.Fatalf("FindAndSaveAll() error = %v", err)
	}
	if len(result.ExchangeIDs) != maxSearchResults {
		t.Fatalf("results = %d, want %d", len(result.ExchangeIDs), maxSearchResults)
	}
	if repository.savedExchanges[0].Participants[1].GivesItemID != best.ItemID {
		t.Fatalf("first saved exchange = %+v, want best candidate beyond output limit", repository.savedExchanges[0])
	}
}

func TestFindAndSaveRankingStatsError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("stats unavailable")
	nodes := makeNodes(2)
	repository := &fakeRepository{
		neighbors:        cycleGraph(nodes),
		searchStatsError: databaseError,
	}

	_, err := New(repository).FindAndSave(context.Background(), nodes[0])
	if !errors.Is(err, databaseError) {
		t.Fatalf("FindAndSave() error = %v, want wrapped %v", err, databaseError)
	}
	if repository.saveCalls != 0 {
		t.Fatalf("SaveExchange() calls = %d, want 0", repository.saveCalls)
	}
}

func TestFindAndSaveAllSkipsDuplicateAndStaleCandidates(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	alternatives := []exchangemodel.Node{testNode(2), testNode(3), testNode(4)}
	neighbors := map[uuid.UUID][]exchangemodel.Node{start.ItemID: alternatives}
	for _, alternative := range alternatives {
		neighbors[alternative.ItemID] = []exchangemodel.Node{start}
	}
	repository := &fakeRepository{
		neighbors:       neighbors,
		savedExchangeID: uuid.New(),
		saveErrors:      []error{ErrDuplicateExchange, ErrStaleSearchResult, nil},
	}

	result, err := New(repository).FindAndSaveAll(context.Background(), start)
	if err != nil {
		t.Fatalf("FindAndSaveAll() error = %v", err)
	}
	if !result.Found || len(result.ExchangeIDs) != 1 {
		t.Fatalf("FindAndSaveAll() = %+v, want one saved alternative", result)
	}
	if repository.saveCalls != 3 {
		t.Fatalf("SaveExchange() calls = %d, want 3", repository.saveCalls)
	}
}

func TestFindAndSaveAllStopsAtMaximumResults(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	alternatives := make([]exchangemodel.Node, maxSearchResults+2)
	neighbors := make(map[uuid.UUID][]exchangemodel.Node, len(alternatives)+1)
	for index := range alternatives {
		alternatives[index] = testNode(byte(index + 2))
		neighbors[alternatives[index].ItemID] = []exchangemodel.Node{start}
	}
	neighbors[start.ItemID] = alternatives
	repository := &fakeRepository{neighbors: neighbors, savedExchangeID: uuid.New()}

	result, err := New(repository).FindAndSaveAll(context.Background(), start)
	if err != nil {
		t.Fatalf("FindAndSaveAll() error = %v", err)
	}
	if len(result.ExchangeIDs) != maxSearchResults || repository.saveCalls != maxSearchResults {
		t.Fatalf("results=%d saves=%d, want %d", len(result.ExchangeIDs), repository.saveCalls, maxSearchResults)
	}
}

func TestScheduleSearchEnqueuesItemJob(t *testing.T) {
	t.Parallel()

	queue := &fakeSearchQueue{}
	start := testNode(1)
	service := newWithDependencies(&fakeRepository{}, queue, &fakeLogger{})

	if err := service.ScheduleSearch(context.Background(), start); err != nil {
		t.Fatalf("ScheduleSearch() error = %v", err)
	}
	if queue.calls != 1 || len(queue.job.Nodes) != 1 || queue.job.Nodes[0] != start {
		t.Fatalf("queued job = %+v, calls = %d", queue.job, queue.calls)
	}
}

func TestScheduleSearchWrapsQueueError(t *testing.T) {
	t.Parallel()

	queueError := errors.New("queue full")
	queue := &fakeSearchQueue{err: queueError}
	service := newWithDependencies(&fakeRepository{}, queue, &fakeLogger{})

	if err := service.ScheduleSearch(context.Background(), testNode(1)); !errors.Is(err, queueError) {
		t.Fatalf("ScheduleSearch() error = %v, want wrapped %v", err, queueError)
	}
}

func TestProcessRecoveryJobExcludesCancelledComposition(t *testing.T) {
	t.Parallel()

	start := testNode(1)
	refused := testNode(2)
	alternative := testNode(3)
	repository := &fakeRepository{
		neighbors: map[uuid.UUID][]exchangemodel.Node{
			start.ItemID:       {refused, alternative},
			refused.ItemID:     {start},
			alternative.ItemID: {start},
		},
		savedExchangeID: uuid.New(),
	}
	job := exchangesearch.NewRecoveryJob(
		[]exchangemodel.Node{start},
		compositionKey([]exchangemodel.Node{start, refused}),
	)

	if err := New(repository).ProcessSearchJob(context.Background(), job); err != nil {
		t.Fatalf("ProcessSearchJob() error = %v", err)
	}
	if len(repository.savedExchanges) != 1 ||
		repository.savedExchanges[0].Participants[1].GivesItemID != alternative.ItemID {
		t.Fatalf("saved exchanges = %+v, want only alternative composition", repository.savedExchanges)
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

	// Читатель доезжает до запроса: по нему считается счётчик непрочитанного.
	if repository.detailsUserID != userID {
		t.Fatalf("GetByID() user ID = %s, want %s", repository.detailsUserID, userID)
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

func TestDeclineParticipationRecoversWithDifferentCycle(t *testing.T) {
	t.Parallel()

	original := makeNodes(3)
	alternative := testNode(4)
	repository := &fakeRepository{
		declineRecovery: original,
		saveErrors:      []error{ErrDuplicateExchange, nil},
		neighbors: map[uuid.UUID][]exchangemodel.Node{
			original[0].ItemID: {original[1]},
			original[1].ItemID: {original[2], alternative},
			original[2].ItemID: {original[0]},
			alternative.ItemID: {original[0]},
		},
	}

	if err := New(repository).DeclineParticipation(context.Background(), uuid.New(), original[0].OwnerID); err != nil {
		t.Fatalf("DeclineParticipation() error = %v", err)
	}
	if repository.saveCalls != 2 {
		t.Fatalf("SaveExchange() calls = %d, want duplicate plus alternative", repository.saveCalls)
	}

	wantItems := map[uuid.UUID]bool{
		original[0].ItemID: true,
		original[1].ItemID: true,
		alternative.ItemID: true,
	}
	for _, participant := range repository.savedExchange.Participants {
		if !wantItems[participant.GivesItemID] {
			t.Fatalf("recovered exchange contains unexpected item %s", participant.GivesItemID)
		}
		delete(wantItems, participant.GivesItemID)
	}
	if len(wantItems) != 0 {
		t.Fatalf("recovered exchange misses items: %v", wantItems)
	}
}

func TestDeclineParticipationDoesNotRecreateSameCycle(t *testing.T) {
	t.Parallel()

	// Подпись уникальна только среди открытых обменов, поэтому отменённый состав база
	// больше не отклоняет. Не переподставить его — задача самого перепоиска.
	nodes := makeNodes(3)
	repository := &fakeRepository{
		declineRecovery:    nodes,
		declineComposition: compositionKey(nodes),
		neighbors:          cycleGraph(nodes),
	}

	if err := New(repository).DeclineParticipation(context.Background(), uuid.New(), nodes[0].OwnerID); err != nil {
		t.Fatalf("DeclineParticipation() error = %v", err)
	}
	if repository.saveCalls != 0 {
		t.Fatalf("SaveExchange() calls = %d, want the cancelled cycle left alone", repository.saveCalls)
	}
}

func TestDeclineParticipationKeepsSuccessWhenRecoveryFails(t *testing.T) {
	t.Parallel()

	nodes := makeNodes(2)
	databaseError := errors.New("database unavailable")
	logger := &fakeLogger{}
	repository := &fakeRepository{
		declineRecovery: nodes,
		errors:          map[uuid.UUID]error{nodes[0].ItemID: databaseError},
	}

	err := newWithDependencies(repository, nil, logger).
		DeclineParticipation(context.Background(), uuid.New(), nodes[0].OwnerID)
	if err != nil {
		t.Fatalf("DeclineParticipation() error = %v, want successful decline", err)
	}
	if logger.calls == 0 {
		t.Fatal("recovery error was not logged")
	}
}

func TestDeclineParticipationSchedulesRecoveryJob(t *testing.T) {
	t.Parallel()

	nodes := makeNodes(3)
	cancelledComposition := compositionKey(nodes)
	repository := &fakeRepository{
		declineRecovery:    nodes,
		declineComposition: cancelledComposition,
	}
	queue := &fakeSearchQueue{}
	service := newWithDependencies(repository, queue, &fakeLogger{})

	if err := service.DeclineParticipation(context.Background(), uuid.New(), nodes[0].OwnerID); err != nil {
		t.Fatalf("DeclineParticipation() error = %v", err)
	}
	if queue.calls != 1 || len(queue.job.Nodes) != len(nodes) {
		t.Fatalf("recovery queue calls = %d, job = %+v", queue.calls, queue.job)
	}
	if len(queue.job.ExcludedCompositions) != 1 ||
		queue.job.ExcludedCompositions[0] != cancelledComposition {
		t.Fatalf("excluded compositions = %v, want [%s]", queue.job.ExcludedCompositions, cancelledComposition)
	}
	if repository.saveCalls != 0 {
		t.Fatalf("synchronous SaveExchange() calls = %d, want 0", repository.saveCalls)
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

func TestPostMessageTrimsBodyAndReturnsMessage(t *testing.T) {
	t.Parallel()

	exchangeID := uuid.New()
	userID := uuid.New()
	stored := exchangemodel.Message{ID: uuid.New(), Kind: "text"}
	repository := &fakeRepository{
		accessStatus:        "proposed",
		accessIsParticipant: true,
		createdMessage:      stored,
	}

	message, err := New(repository).PostMessage(
		context.Background(),
		exchangeID,
		userID,
		"  забираю в субботу  ",
	)
	if err != nil {
		t.Fatalf("PostMessage() error = %v", err)
	}

	if message.ID != stored.ID {
		t.Fatalf("PostMessage() message = %+v, want %+v", message, stored)
	}
	if repository.createdBody != "забираю в субботу" {
		t.Fatalf("stored body = %q, want trimmed", repository.createdBody)
	}
	if repository.accessExchangeID != exchangeID || repository.accessUserID != userID {
		t.Fatalf(
			"ExchangeAccess() args = (%s, %s), want (%s, %s)",
			repository.accessExchangeID,
			repository.accessUserID,
			exchangeID,
			userID,
		)
	}
	if repository.createdAuthorID != userID {
		t.Fatalf("message author = %s, want %s", repository.createdAuthorID, userID)
	}
}

func TestPostMessageAllowsConfirmedExchange(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{accessStatus: "confirmed", accessIsParticipant: true}

	if _, err := New(repository).PostMessage(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"встречаемся у метро",
	); err != nil {
		t.Fatalf("PostMessage() error = %v, want nil for confirmed exchange", err)
	}
}

func TestPostMessageRejectsInvalidBody(t *testing.T) {
	t.Parallel()

	bodies := map[string]string{
		"empty":      "",
		"whitespace": "   \n\t ",
		"too long":   strings.Repeat("я", maxMessageLength+1),
		// Postgres не принимает NUL в text: без проверки это 500, а не 400.
		"nul byte": "при\x00вет",
	}

	for name, body := range bodies {
		body := body
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repository := &fakeRepository{accessStatus: "proposed", accessIsParticipant: true}

			_, err := New(repository).PostMessage(context.Background(), uuid.New(), uuid.New(), body)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("PostMessage() error = %v, want ErrValidation", err)
			}
			if repository.createMessageCalls != 0 {
				t.Fatal("PostMessage() reached the repository with an invalid body")
			}
		})
	}
}

func TestPostMessageAcceptsBodyAtLengthLimit(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{accessStatus: "proposed", accessIsParticipant: true}

	if _, err := New(repository).PostMessage(
		context.Background(),
		uuid.New(),
		uuid.New(),
		strings.Repeat("я", maxMessageLength),
	); err != nil {
		t.Fatalf("PostMessage() error = %v, want nil at the length limit", err)
	}
}

func TestPostMessageRejectsNonParticipant(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{accessStatus: "proposed", accessIsParticipant: false}

	_, err := New(repository).PostMessage(context.Background(), uuid.New(), uuid.New(), "привет")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("PostMessage() error = %v, want ErrForbidden", err)
	}
	if repository.createMessageCalls != 0 {
		t.Fatal("PostMessage() wrote a message for a non-participant")
	}
}

func TestPostMessageRejectsClosedExchange(t *testing.T) {
	t.Parallel()

	for _, status := range []string{"cancelled", "completed"} {
		status := status
		t.Run(status, func(t *testing.T) {
			t.Parallel()

			repository := &fakeRepository{accessStatus: status, accessIsParticipant: true}

			_, err := New(repository).PostMessage(context.Background(), uuid.New(), uuid.New(), "привет")
			if !errors.Is(err, ErrConflict) {
				t.Fatalf("PostMessage() error = %v, want ErrConflict", err)
			}
			if repository.createMessageCalls != 0 {
				t.Fatalf("PostMessage() wrote a message into a %s exchange", status)
			}
		})
	}
}

func TestPostMessageWrapsAccessError(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{accessErr: ErrNotFound}

	_, err := New(repository).PostMessage(context.Background(), uuid.New(), uuid.New(), "привет")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("PostMessage() error = %v, want ErrNotFound", err)
	}
}

func TestListMessagesReturnsThread(t *testing.T) {
	t.Parallel()

	exchangeID := uuid.New()
	thread := []exchangemodel.Message{{ID: uuid.New(), Kind: "text"}}
	repository := &fakeRepository{accessIsParticipant: true, accessStatus: "cancelled", threadMessages: thread}

	messages, err := New(repository).ListMessages(context.Background(), exchangeID, uuid.New())
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}

	if !reflect.DeepEqual(messages, thread) {
		t.Fatalf("ListMessages() = %+v, want %+v", messages, thread)
	}
	if repository.threadExchangeID != exchangeID {
		t.Fatalf("ListMessages() exchange ID = %s, want %s", repository.threadExchangeID, exchangeID)
	}
}

// Чтение треда не должно двигать отметку: иначе фоновый опрос гасил бы счётчик
// непрочитанного вслепую. За отметку отвечает только MarkThreadRead.
func TestListMessagesLeavesTheReadMarkAlone(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{accessIsParticipant: true}

	if _, err := New(repository).ListMessages(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}

	if repository.readExchangeID != uuid.Nil {
		t.Fatal("ListMessages() moved the read mark")
	}
}

func TestListMessagesRejectsNonParticipant(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{accessIsParticipant: false}

	if _, err := New(repository).ListMessages(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("ListMessages() error = %v, want ErrForbidden", err)
	}
}

func TestMarkThreadReadPassesTheLastMessage(t *testing.T) {
	t.Parallel()

	exchangeID, userID, lastMessageID := uuid.New(), uuid.New(), uuid.New()
	// Закрытый обмен: дочитывать историю сделки можно, статус тут ничего не решает.
	repository := &fakeRepository{accessIsParticipant: true, accessStatus: "completed"}

	err := New(repository).MarkThreadRead(context.Background(), exchangeID, userID, lastMessageID)
	if err != nil {
		t.Fatalf("MarkThreadRead() error = %v", err)
	}

	if repository.readExchangeID != exchangeID ||
		repository.readUserID != userID ||
		repository.readLastMessageID != lastMessageID {
		t.Fatalf(
			"MarkMessagesRead() args = (%s, %s, %s), want (%s, %s, %s)",
			repository.readExchangeID, repository.readUserID, repository.readLastMessageID,
			exchangeID, userID, lastMessageID,
		)
	}
}

func TestMarkThreadReadRejectsNonParticipant(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{accessIsParticipant: false}

	err := New(repository).MarkThreadRead(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("MarkThreadRead() error = %v, want ErrForbidden", err)
	}
	if repository.readExchangeID != uuid.Nil {
		t.Fatal("thread was marked read for a non-participant")
	}
}

func TestMarkThreadReadWrapsAccessError(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{accessErr: ErrNotFound}

	err := New(repository).MarkThreadRead(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("MarkThreadRead() error = %v, want ErrNotFound", err)
	}
}

func TestListMessagesWrapsRepositoryError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	repository := &fakeRepository{accessIsParticipant: true, threadErr: databaseError}

	if _, err := New(repository).ListMessages(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, databaseError) {
		t.Fatalf("ListMessages() error = %v, want wrapped %v", err, databaseError)
	}
}

type fakeRepository struct {
	neighbors           map[uuid.UUID][]exchangemodel.Node
	errors              map[uuid.UUID]error
	calls               int
	savedExchange       exchangemodel.Exchange
	savedExchanges      []exchangemodel.Exchange
	savedExchangeID     uuid.UUID
	saveErr             error
	saveErrors          []error
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
	declineRecovery     []exchangemodel.Node
	declineComposition  string
	declineErr          error
	adminCancel         fakeAdminCancellation
	adminDelivery       fakeAdminDelivery
	blockConflicts      map[uuid.UUID]bool
	blockConflictErrors map[uuid.UUID]error
	blockChecks         map[uuid.UUID][]uuid.UUID
	searchStats         map[uuid.UUID]exchangemodel.SearchUserStats
	searchStatsError    error
	searchStatsUserIDs  []uuid.UUID
	searchFilters       map[uuid.UUID]exchangemodel.SearchItemFilters
	searchFiltersError  error
	completedExchangeID uuid.UUID
	completedUserID     uuid.UUID
	completeErr         error
	pickupItemID        uuid.UUID
	pickupOwnerID       uuid.UUID
	pickupPointID       uuid.UUID
	pickupErr           error
	accessStatus        string
	accessIsParticipant bool
	accessErr           error
	accessExchangeID    uuid.UUID
	accessUserID        uuid.UUID
	createdMessage      exchangemodel.Message
	createdBody         string
	createdAuthorID     uuid.UUID
	createMessageErr    error
	createMessageCalls  int
	threadMessages      []exchangemodel.Message
	threadExchangeID    uuid.UUID
	threadErr           error
	detailsUserID       uuid.UUID
	readExchangeID      uuid.UUID
	readUserID          uuid.UUID
	readLastMessageID   uuid.UUID
	readErr             error
}

type fakeSearchQueue struct {
	job   exchangesearch.Job
	calls int
	err   error
}

func (f *fakeSearchQueue) Enqueue(job exchangesearch.Job) (bool, error) {
	f.job = job
	f.calls++
	return f.err == nil, f.err
}

func (f *fakeRepository) ExchangeAccess(
	_ context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) (string, bool, error) {
	f.accessExchangeID = exchangeID
	f.accessUserID = userID
	return f.accessStatus, f.accessIsParticipant, f.accessErr
}

func (f *fakeRepository) CreateMessage(
	_ context.Context,
	exchangeID uuid.UUID,
	authorID uuid.UUID,
	body string,
) (exchangemodel.Message, error) {
	f.createMessageCalls++
	f.requestedExchangeID = exchangeID
	f.createdAuthorID = authorID
	f.createdBody = body
	return f.createdMessage, f.createMessageErr
}

func (f *fakeRepository) ListMessages(
	_ context.Context,
	exchangeID uuid.UUID,
) ([]exchangemodel.Message, error) {
	f.threadExchangeID = exchangeID
	return f.threadMessages, f.threadErr
}

type fakeLogger struct {
	calls int
}

func (f *fakeLogger) Printf(string, ...any) {
	f.calls++
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

func (f *fakeRepository) HasUserBlockConflict(
	_ context.Context,
	candidateUserID uuid.UUID,
	pathUserIDs []uuid.UUID,
) (bool, error) {
	if f.blockChecks == nil {
		f.blockChecks = make(map[uuid.UUID][]uuid.UUID)
	}
	f.blockChecks[candidateUserID] = append([]uuid.UUID(nil), pathUserIDs...)
	if err := f.blockConflictErrors[candidateUserID]; err != nil {
		return false, err
	}
	return f.blockConflicts[candidateUserID], nil
}

func (f *fakeRepository) GetSearchUserStats(
	_ context.Context,
	userIDs []uuid.UUID,
) (map[uuid.UUID]exchangemodel.SearchUserStats, error) {
	f.searchStatsUserIDs = append([]uuid.UUID(nil), userIDs...)
	return f.searchStats, f.searchStatsError
}

func (f *fakeRepository) GetSearchItemFilters(
	_ context.Context,
	itemIDs []uuid.UUID,
) (map[uuid.UUID]exchangemodel.SearchItemFilters, error) {
	if f.searchFilters != nil {
		return f.searchFilters, f.searchFiltersError
	}
	filters := make(map[uuid.UUID]exchangemodel.SearchItemFilters, len(itemIDs))
	for _, itemID := range itemIDs {
		filters[itemID] = exchangemodel.SearchItemFilters{ItemID: itemID, MaxChainLength: maxParticipants, PreferReliableParticipants: true}
	}
	return filters, f.searchFiltersError
}

func (f *fakeRepository) SaveExchange(
	_ context.Context,
	exchange exchangemodel.Exchange,
) (uuid.UUID, error) {
	f.saveCalls++
	f.savedExchange = exchange
	f.savedExchanges = append(f.savedExchanges, exchange)
	if len(f.saveErrors) >= f.saveCalls {
		return f.savedExchangeID, f.saveErrors[f.saveCalls-1]
	}
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
	userID uuid.UUID,
) (exchangemodel.Details, error) {
	f.requestedExchangeID = exchangeID
	f.detailsUserID = userID
	return f.exchangeDetails, f.getErr
}

func (f *fakeRepository) MarkMessagesRead(
	_ context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
	lastMessageID uuid.UUID,
) error {
	f.readExchangeID = exchangeID
	f.readUserID = userID
	f.readLastMessageID = lastMessageID
	return f.readErr
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
) ([]exchangemodel.Node, string, error) {
	f.declinedExchangeID = exchangeID
	f.declinedUserID = userID
	return append([]exchangemodel.Node(nil), f.declineRecovery...), f.declineComposition, f.declineErr
}

func (f *fakeRepository) CancelByAdmin(
	_ context.Context,
	exchangeID uuid.UUID,
	adminID uuid.UUID,
) ([]exchangemodel.Node, string, error) {
	f.adminCancel.exchangeID = exchangeID
	f.adminCancel.adminID = adminID
	return append([]exchangemodel.Node(nil), f.adminCancel.recovery...),
		f.adminCancel.composition,
		f.adminCancel.err
}

func (f *fakeRepository) MarkDeliveredByAdmin(
	_ context.Context,
	exchangeID uuid.UUID,
	adminID uuid.UUID,
) error {
	f.adminDelivery.exchangeID = exchangeID
	f.adminDelivery.adminID = adminID
	return f.adminDelivery.err
}

type fakeAdminCancellation struct {
	exchangeID  uuid.UUID
	adminID     uuid.UUID
	recovery    []exchangemodel.Node
	composition string
	err         error
}

type fakeAdminDelivery struct {
	exchangeID uuid.UUID
	adminID    uuid.UUID
	err        error
}

func (f *fakeRepository) RecordItemPickup(
	_ context.Context,
	itemID uuid.UUID,
	ownerID uuid.UUID,
	pickupPointID uuid.UUID,
) error {
	f.pickupItemID = itemID
	f.pickupOwnerID = ownerID
	f.pickupPointID = pickupPointID

	return f.pickupErr
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
