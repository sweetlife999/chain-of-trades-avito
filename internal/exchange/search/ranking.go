package search

import (
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

const (
	ratingWeight      = 0.50
	reliabilityWeight = 0.30
	experienceWeight  = 0.10
	compactnessWeight = 0.10

	neutralRating       = 3.0
	maximumRating       = 5.0
	experienceDealsCap  = 20.0
	minimumParticipants = 2
	maximumParticipants = 5
	scoreEpsilon        = 1e-9
)

type rankedCycle struct {
	nodes []exchangemodel.Node
	score float64
	key   string
}

// RankCycles orders exchange alternatives from the most useful to the least useful.
// It returns copies of the input paths, so callers may safely reuse the DFS result.
func RankCycles(
	cycles [][]exchangemodel.Node,
	stats map[uuid.UUID]exchangemodel.SearchUserStats,
) [][]exchangemodel.Node {
	return RankCyclesWithFilters(cycles, stats, nil)
}

func RankCyclesWithFilters(
	cycles [][]exchangemodel.Node,
	stats map[uuid.UUID]exchangemodel.SearchUserStats,
	filters map[uuid.UUID]exchangemodel.SearchItemFilters,
) [][]exchangemodel.Node {
	ranked := make([]rankedCycle, len(cycles))
	for index, cycle := range cycles {
		cycleCopy := append([]exchangemodel.Node(nil), cycle...)
		ranked[index] = rankedCycle{
			nodes: cycleCopy,
			score: cycleScore(cycleCopy, stats, prefersReliable(cycleCopy, filters)),
			key:   cycleCompositionKey(cycleCopy),
		}
	}

	sort.SliceStable(ranked, func(first, second int) bool {
		difference := ranked[first].score - ranked[second].score
		if math.Abs(difference) > scoreEpsilon {
			return difference > 0
		}

		return ranked[first].key < ranked[second].key
	})

	result := make([][]exchangemodel.Node, len(ranked))
	for index := range ranked {
		result[index] = ranked[index].nodes
	}

	return result
}

func cycleScore(
	cycle []exchangemodel.Node,
	stats map[uuid.UUID]exchangemodel.SearchUserStats,
	preferReliable bool,
) float64 {
	if len(cycle) == 0 {
		return 0
	}

	var rating, reliability, experience float64
	for _, node := range cycle {
		userStats, found := stats[node.OwnerID]
		if !found {
			userStats.Rating = neutralRating
		}

		completed := max(float64(userStats.DealsCompleted), 0)
		broken := max(float64(userStats.DealsBroken), 0)
		rating += clamp(userStats.Rating, 0, maximumRating) / maximumRating
		// Laplace smoothing keeps a new user neutral instead of treating 0/0 as
		// either perfect or unreliable.
		reliability += (completed + 1) / (completed + broken + 2)
		experience += math.Min(completed+broken, experienceDealsCap) / experienceDealsCap
	}

	participants := float64(len(cycle))
	rating /= participants
	reliability /= participants
	experience /= participants

	compactness := 1 - (participants-minimumParticipants)/
		(maximumParticipants-minimumParticipants)
	compactness = clamp(compactness, 0, 1)

	reliabilityScore := 0.0
	if preferReliable {
		reliabilityScore = reliability * reliabilityWeight
	}
	return rating*ratingWeight +
		reliabilityScore +
		experience*experienceWeight +
		compactness*compactnessWeight
}

func prefersReliable(
	cycle []exchangemodel.Node,
	filters map[uuid.UUID]exchangemodel.SearchItemFilters,
) bool {
	if filters == nil {
		return true
	}
	for _, node := range cycle {
		filter, found := filters[node.ItemID]
		if !found || filter.PreferReliableParticipants {
			return true
		}
	}
	return false
}

func cycleCompositionKey(cycle []exchangemodel.Node) string {
	itemIDs := make([]string, len(cycle))
	for index, node := range cycle {
		itemIDs[index] = node.ItemID.String()
	}
	sort.Strings(itemIDs)

	return strings.Join(itemIDs, "|")
}

func clamp(value, minimum, maximum float64) float64 {
	return math.Min(math.Max(value, minimum), maximum)
}
