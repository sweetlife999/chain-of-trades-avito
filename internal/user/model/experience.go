package model

const (
	// ExperiencePerCompletedDeal keeps the progression tied to the only action
	// that cannot be rolled back or farmed by accepting and cancelling a deal.
	ExperiencePerCompletedDeal int32 = 100
	MaxExperienceLevel         int32 = 5
)

var experienceLevelDeals = [...]int32{0, 1, 3, 6, 10}

// Experience describes a derived user progression. It is deliberately not
// stored in the database: deals_completed remains the single source of truth,
// so existing users are backfilled automatically and the values cannot drift.
type Experience struct {
	Level           int32
	MaxLevel        int32
	TotalXP         int64
	CurrentLevelXP  int32
	NextLevelXP     int32
	ProgressPercent int32
	DealsToNext     int32
	IsMaxLevel      bool
}

// ExperienceFromCompletedDeals calculates a stable five-level progression.
func ExperienceFromCompletedDeals(completedDeals int32) Experience {
	if completedDeals < 0 {
		completedDeals = 0
	}

	levelIndex := 0
	for index, requiredDeals := range experienceLevelDeals {
		if completedDeals < requiredDeals {
			break
		}
		levelIndex = index
	}

	totalXP := int64(completedDeals) * int64(ExperiencePerCompletedDeal)
	level := int32(levelIndex + 1)
	isMaxLevel := levelIndex == len(experienceLevelDeals)-1
	if isMaxLevel {
		return Experience{
			Level:           level,
			MaxLevel:        MaxExperienceLevel,
			TotalXP:         totalXP,
			ProgressPercent: 100,
			IsMaxLevel:      true,
		}
	}

	currentLevelDeals := experienceLevelDeals[levelIndex]
	nextLevelDeals := experienceLevelDeals[levelIndex+1]
	completedWithinLevel := completedDeals - currentLevelDeals
	dealsRequiredWithinLevel := nextLevelDeals - currentLevelDeals

	return Experience{
		Level:           level,
		MaxLevel:        MaxExperienceLevel,
		TotalXP:         totalXP,
		CurrentLevelXP:  completedWithinLevel * ExperiencePerCompletedDeal,
		NextLevelXP:     dealsRequiredWithinLevel * ExperiencePerCompletedDeal,
		ProgressPercent: completedWithinLevel * 100 / dealsRequiredWithinLevel,
		DealsToNext:     nextLevelDeals - completedDeals,
		IsMaxLevel:      false,
	}
}
