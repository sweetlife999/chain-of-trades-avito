package dto

import (
	"testing"

	"github.com/google/uuid"

	usermodel "github.com/sweetlife999/chain-of-trades-avito/internal/user/model"
)

func TestFromModelIncludesExperience(t *testing.T) {
	t.Parallel()

	response := FromModel(usermodel.User{
		ID:             uuid.New(),
		Nickname:       "tester",
		DealsCompleted: 4,
	})

	if response.Experience.Level != 3 || response.Experience.TotalXP != 400 {
		t.Fatalf("experience = %+v, want level 3 and 400 XP", response.Experience)
	}
	if response.Experience.DealsToNext != 2 || response.Experience.ProgressPercent != 33 {
		t.Fatalf("experience progress = %+v, want 2 deals and 33%%", response.Experience)
	}
}
