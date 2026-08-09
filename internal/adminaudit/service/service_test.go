package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	adminauditmodel "github.com/sweetlife999/chain-of-trades-avito/internal/adminaudit/model"
)

type fakeRepository struct {
	state   adminauditmodel.UserBlockState
	entries []adminauditmodel.Entry
	total   int64
	err     error
	adminID uuid.UUID
	userID  uuid.UUID
	filter  adminauditmodel.Filter
}

func (f *fakeRepository) BlockUser(_ context.Context, adminID, userID uuid.UUID) (adminauditmodel.UserBlockState, error) {
	f.adminID, f.userID = adminID, userID
	return f.state, f.err
}
func (f *fakeRepository) UnblockUser(_ context.Context, adminID, userID uuid.UUID) (adminauditmodel.UserBlockState, error) {
	f.adminID, f.userID = adminID, userID
	return f.state, f.err
}
func (f *fakeRepository) List(_ context.Context, filter adminauditmodel.Filter) ([]adminauditmodel.Entry, error) {
	f.filter = filter
	return f.entries, f.err
}
func (f *fakeRepository) Count(_ context.Context, filter adminauditmodel.Filter) (int64, error) {
	f.filter = filter
	return f.total, f.err
}

func TestBlockUserRejectsSelf(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	_, err := New(&fakeRepository{}).BlockUser(context.Background(), id, id)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("BlockUser() error = %v", err)
	}
}

func TestBlockUserDelegates(t *testing.T) {
	t.Parallel()
	adminID, userID := uuid.New(), uuid.New()
	repository := &fakeRepository{state: adminauditmodel.UserBlockState{ID: userID, IsBlocked: true}}
	state, err := New(repository).BlockUser(context.Background(), adminID, userID)
	if err != nil {
		t.Fatalf("BlockUser() error = %v", err)
	}
	if repository.adminID != adminID || repository.userID != userID || !state.IsBlocked {
		t.Fatalf("state/repository = %+v/%+v", state, repository)
	}
}

func TestListValidatesAndReturnsPage(t *testing.T) {
	t.Parallel()
	from, to := time.Now(), time.Now().Add(time.Hour)
	repository := &fakeRepository{entries: []adminauditmodel.Entry{{ID: uuid.New()}}, total: 1}
	page, err := New(repository).List(context.Background(), adminauditmodel.Filter{
		Action: " user_blocked ", From: &from, To: &to,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if page.Limit != DefaultLimit || page.Total != 1 || repository.filter.Action != "user_blocked" {
		t.Fatalf("page/filter = %+v/%+v", page, repository.filter)
	}
}

func TestListRejectsInvalidFilters(t *testing.T) {
	t.Parallel()
	from, to := time.Now(), time.Now().Add(-time.Hour)
	tests := []adminauditmodel.Filter{
		{Action: "unknown"}, {Limit: MaxLimit + 1}, {Limit: 1, Offset: -1}, {From: &from, To: &to},
	}
	for _, filter := range tests {
		_, err := New(&fakeRepository{}).List(context.Background(), filter)
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("List(%+v) error = %v", filter, err)
		}
	}
}
