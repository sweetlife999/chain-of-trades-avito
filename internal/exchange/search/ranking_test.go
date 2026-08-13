package search

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

func TestRankCyclesPrefersHigherRating(t *testing.T) {
	t.Parallel()

	lowRated := rankingNode(1)
	highRated := rankingNode(2)
	cycles := [][]exchangemodel.Node{{lowRated, rankingNode(3)}, {highRated, rankingNode(4)}}
	stats := neutralStats(cycles)
	stats[lowRated.OwnerID] = searchStats(lowRated.OwnerID, 10, 1, 2.0)
	stats[highRated.OwnerID] = searchStats(highRated.OwnerID, 10, 1, 5.0)

	ranked := RankCycles(cycles, stats)

	assertFirstCycleContains(t, ranked, highRated.ItemID)
}

func TestRankCyclesPrefersReliableParticipants(t *testing.T) {
	t.Parallel()

	unreliable := rankingNode(1)
	reliable := rankingNode(2)
	cycles := [][]exchangemodel.Node{{unreliable, rankingNode(3)}, {reliable, rankingNode(4)}}
	stats := neutralStats(cycles)
	stats[unreliable.OwnerID] = searchStats(unreliable.OwnerID, 2, 8, neutralRating)
	stats[reliable.OwnerID] = searchStats(reliable.OwnerID, 8, 2, neutralRating)

	ranked := RankCycles(cycles, stats)

	assertFirstCycleContains(t, ranked, reliable.ItemID)
}

func TestRankCyclesCanDisableReliabilityPreference(t *testing.T) {
	t.Parallel()
	unreliable := rankingNode(1)
	reliable := rankingNode(2)
	first := []exchangemodel.Node{unreliable, rankingNode(3)}
	second := []exchangemodel.Node{reliable, rankingNode(4)}
	stats := neutralStats([][]exchangemodel.Node{first, second})
	stats[unreliable.OwnerID] = searchStats(unreliable.OwnerID, 2, 8, neutralRating)
	stats[reliable.OwnerID] = searchStats(reliable.OwnerID, 8, 2, neutralRating)
	filters := map[uuid.UUID]exchangemodel.SearchItemFilters{}
	for _, cycle := range [][]exchangemodel.Node{first, second} {
		for _, node := range cycle {
			filters[node.ItemID] = exchangemodel.SearchItemFilters{ItemID: node.ItemID}
		}
	}

	ranked := RankCyclesWithFilters([][]exchangemodel.Node{first, second}, stats, filters)
	wantFirst := first
	if cycleCompositionKey(second) < cycleCompositionKey(first) {
		wantFirst = second
	}
	if cycleCompositionKey(ranked[0]) != cycleCompositionKey(wantFirst) {
		t.Fatalf("reliability affected disabled preference: first = %q, want %q", cycleCompositionKey(ranked[0]), cycleCompositionKey(wantFirst))
	}
}

func TestRankCyclesPrefersShorterCycleWhenUsersAreEqual(t *testing.T) {
	t.Parallel()

	shortCycle := []exchangemodel.Node{rankingNode(1), rankingNode(2)}
	longCycle := []exchangemodel.Node{rankingNode(3), rankingNode(4), rankingNode(5), rankingNode(6)}
	cycles := [][]exchangemodel.Node{longCycle, shortCycle}

	ranked := RankCycles(cycles, neutralStats(cycles))

	if len(ranked[0]) != len(shortCycle) {
		t.Fatalf("first ranked cycle length = %d, want %d", len(ranked[0]), len(shortCycle))
	}
}

func TestRankCyclesTreatsMissingStatsAsNeutral(t *testing.T) {
	t.Parallel()

	missingStatsCycle := []exchangemodel.Node{rankingNode(1), rankingNode(2)}
	explicitNeutralCycle := []exchangemodel.Node{rankingNode(3), rankingNode(4)}
	stats := neutralStats([][]exchangemodel.Node{explicitNeutralCycle})

	missingScore := cycleScore(missingStatsCycle, stats, true)
	neutralScore := cycleScore(explicitNeutralCycle, stats, true)
	if missingScore != neutralScore {
		t.Fatalf("missing stats score = %f, explicit neutral score = %f", missingScore, neutralScore)
	}
}

func TestRankCyclesUsesStableCompositionTieBreak(t *testing.T) {
	t.Parallel()

	firstByKey := []exchangemodel.Node{rankingNode(1), rankingNode(2)}
	secondByKey := []exchangemodel.Node{rankingNode(3), rankingNode(4)}
	cycles := [][]exchangemodel.Node{secondByKey, firstByKey}

	ranked := RankCycles(cycles, neutralStats(cycles))

	if cycleCompositionKey(ranked[0]) != cycleCompositionKey(firstByKey) {
		t.Fatalf("first key = %q, want %q", cycleCompositionKey(ranked[0]), cycleCompositionKey(firstByKey))
	}
}

func TestRankCyclesDoesNotMutateInput(t *testing.T) {
	t.Parallel()

	cycles := [][]exchangemodel.Node{{rankingNode(3), rankingNode(4)}, {rankingNode(1), rankingNode(2)}}
	want := make([][]exchangemodel.Node, len(cycles))
	for index := range cycles {
		want[index] = append([]exchangemodel.Node(nil), cycles[index]...)
	}

	RankCycles(cycles, neutralStats(cycles))

	if !reflect.DeepEqual(cycles, want) {
		t.Fatalf("input cycles changed: got %+v, want %+v", cycles, want)
	}
}

func neutralStats(cycles [][]exchangemodel.Node) map[uuid.UUID]exchangemodel.SearchUserStats {
	stats := make(map[uuid.UUID]exchangemodel.SearchUserStats)
	for _, cycle := range cycles {
		for _, node := range cycle {
			stats[node.OwnerID] = searchStats(node.OwnerID, 0, 0, neutralRating)
		}
	}

	return stats
}

func searchStats(userID uuid.UUID, completed, broken int32, rating float64) exchangemodel.SearchUserStats {
	return exchangemodel.SearchUserStats{
		UserID:         userID,
		DealsCompleted: completed,
		DealsBroken:    broken,
		Rating:         rating,
	}
}

func rankingNode(value byte) exchangemodel.Node {
	itemID := uuid.Nil
	itemID[15] = value
	ownerID := uuid.Nil
	ownerID[14] = 1
	ownerID[15] = value

	return exchangemodel.Node{ItemID: itemID, OwnerID: ownerID}
}

func assertFirstCycleContains(t *testing.T, cycles [][]exchangemodel.Node, itemID uuid.UUID) {
	t.Helper()
	if len(cycles) == 0 {
		t.Fatal("RankCycles() returned no cycles")
	}
	for _, node := range cycles[0] {
		if node.ItemID == itemID {
			return
		}
	}

	t.Fatalf("first ranked cycle %+v does not contain item %s", cycles[0], itemID)
}
