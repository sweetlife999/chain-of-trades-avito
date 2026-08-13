package model

import "testing"

func TestExperienceFromCompletedDeals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		completedDeals  int32
		wantLevel       int32
		wantTotalXP     int64
		wantCurrentXP   int32
		wantNextXP      int32
		wantProgress    int32
		wantDealsToNext int32
		wantIsMaxLevel  bool
	}{
		{name: "negative value is safe", completedDeals: -1, wantLevel: 1, wantNextXP: 100, wantDealsToNext: 1},
		{name: "new user", completedDeals: 0, wantLevel: 1, wantNextXP: 100, wantDealsToNext: 1},
		{name: "second level", completedDeals: 1, wantLevel: 2, wantTotalXP: 100, wantNextXP: 200, wantDealsToNext: 2},
		{name: "middle of third level", completedDeals: 4, wantLevel: 3, wantTotalXP: 400, wantCurrentXP: 100, wantNextXP: 300, wantProgress: 33, wantDealsToNext: 2},
		{name: "fourth level boundary", completedDeals: 6, wantLevel: 4, wantTotalXP: 600, wantNextXP: 400, wantDealsToNext: 4},
		{name: "maximum level", completedDeals: 10, wantLevel: 5, wantTotalXP: 1000, wantProgress: 100, wantIsMaxLevel: true},
		{name: "experience continues after maximum", completedDeals: 14, wantLevel: 5, wantTotalXP: 1400, wantProgress: 100, wantIsMaxLevel: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := ExperienceFromCompletedDeals(test.completedDeals)
			if got.Level != test.wantLevel || got.MaxLevel != MaxExperienceLevel {
				t.Fatalf("level = %d/%d, want %d/%d", got.Level, got.MaxLevel, test.wantLevel, MaxExperienceLevel)
			}
			if got.TotalXP != test.wantTotalXP || got.CurrentLevelXP != test.wantCurrentXP || got.NextLevelXP != test.wantNextXP {
				t.Fatalf("xp = total:%d current:%d next:%d, want total:%d current:%d next:%d", got.TotalXP, got.CurrentLevelXP, got.NextLevelXP, test.wantTotalXP, test.wantCurrentXP, test.wantNextXP)
			}
			if got.ProgressPercent != test.wantProgress || got.DealsToNext != test.wantDealsToNext || got.IsMaxLevel != test.wantIsMaxLevel {
				t.Fatalf("progress = %d%%, deals_to_next = %d, max = %t", got.ProgressPercent, got.DealsToNext, got.IsMaxLevel)
			}
		})
	}
}
