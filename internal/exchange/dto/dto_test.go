package dto

import (
	"testing"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

func TestFromModelIncludesCancellationReason(t *testing.T) {
	t.Parallel()

	reason := "confirmed_broken"
	response := FromModel(exchangemodel.Details{
		ID:           uuid.New(),
		Status:       "cancelled",
		CancelReason: &reason,
	})

	if response.CancelReason == nil || *response.CancelReason != reason {
		t.Fatalf("CancelReason = %v, want %q", response.CancelReason, reason)
	}
}

func TestFromModelLeavesActiveCancellationReasonNull(t *testing.T) {
	t.Parallel()

	response := FromModel(exchangemodel.Details{ID: uuid.New(), Status: "proposed"})
	if response.CancelReason != nil {
		t.Fatalf("CancelReason = %v, want nil", response.CancelReason)
	}
}
