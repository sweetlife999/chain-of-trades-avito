package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	usermodel "github.com/sweetlife999/chain-of-trades-avito/internal/user/model"
	userrepository "github.com/sweetlife999/chain-of-trades-avito/internal/user/repository"
)

type fakeUsers struct {
	err    error
	gotID  uuid.UUID
	called bool
}

func (f *fakeUsers) GetByID(_ context.Context, id uuid.UUID) (usermodel.User, error) {
	f.called = true
	f.gotID = id
	return usermodel.User{ID: id}, f.err
}

type fakeExchanges struct {
	items      []exchangemodel.Details
	total      int64
	listErr    error
	countErr   error
	gotUserID  uuid.UUID
	gotStatus  string
	gotLimit   int32
	gotOffset  int32
	listCalled bool
}

func (f *fakeExchanges) ListActiveByUser(
	_ context.Context,
	userID uuid.UUID,
	limit int32,
	offset int32,
) ([]exchangemodel.Details, error) {
	f.listCalled = true
	f.gotUserID = userID
	f.gotLimit = limit
	f.gotOffset = offset
	return f.items, f.listErr
}

func (f *fakeExchanges) CountActiveByUser(context.Context, uuid.UUID) (int64, error) {
	return f.total, f.countErr
}

func (f *fakeExchanges) ListActiveForAdmin(
	_ context.Context,
	status string,
	limit int32,
	offset int32,
) ([]exchangemodel.Details, error) {
	f.listCalled = true
	f.gotStatus = status
	f.gotLimit = limit
	f.gotOffset = offset
	return f.items, f.listErr
}

func (f *fakeExchanges) CountActiveForAdmin(_ context.Context, status string) (int64, error) {
	f.gotStatus = status
	return f.total, f.countErr
}

func TestListActiveByUser(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	exchangeID := uuid.New()
	users := &fakeUsers{}
	exchanges := &fakeExchanges{
		items: []exchangemodel.Details{{ID: exchangeID, Status: "proposed"}},
		total: 3,
	}

	page, err := New(users, exchanges).ListActiveByUser(context.Background(), userID, 10, 20)
	if err != nil {
		t.Fatalf("ListActiveByUser() error = %v", err)
	}
	if users.gotID != userID || exchanges.gotUserID != userID {
		t.Fatalf("user IDs = %v / %v, want %v", users.gotID, exchanges.gotUserID, userID)
	}
	if exchanges.gotLimit != 10 || exchanges.gotOffset != 20 {
		t.Fatalf("pagination = %d/%d", exchanges.gotLimit, exchanges.gotOffset)
	}
	if len(page.Exchanges) != 1 || page.Exchanges[0].ID != exchangeID || page.Total != 3 {
		t.Fatalf("page = %+v", page)
	}
}

func TestListActiveByUserReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	page, err := New(&fakeUsers{}, &fakeExchanges{}).
		ListActiveByUser(context.Background(), uuid.New(), DefaultLimit, 0)
	if err != nil {
		t.Fatalf("ListActiveByUser() error = %v", err)
	}
	if page.Exchanges == nil {
		t.Fatal("Exchanges = nil, want []")
	}
}

func TestListActiveByUserValidationStopsBeforeRepositories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		limit  int32
		offset int32
	}{
		{name: "zero limit", limit: 0},
		{name: "too large limit", limit: MaxLimit + 1},
		{name: "negative offset", limit: DefaultLimit, offset: -1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users := &fakeUsers{}
			exchanges := &fakeExchanges{}
			_, err := New(users, exchanges).
				ListActiveByUser(context.Background(), uuid.New(), test.limit, test.offset)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("error = %v, want %v", err, ErrValidation)
			}
			if users.called || exchanges.listCalled {
				t.Fatal("repositories were called for invalid pagination")
			}
		})
	}
}

func TestListActiveByUserReturnsNotFound(t *testing.T) {
	t.Parallel()

	users := &fakeUsers{err: userrepository.ErrNotFound}
	exchanges := &fakeExchanges{}
	_, err := New(users, exchanges).
		ListActiveByUser(context.Background(), uuid.New(), DefaultLimit, 0)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrNotFound)
	}
	if exchanges.listCalled {
		t.Fatal("exchange repository called for missing user")
	}
}

func TestListActiveByUserWrapsRepositoryErrors(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	_, err := New(&fakeUsers{}, &fakeExchanges{listErr: databaseError}).
		ListActiveByUser(context.Background(), uuid.New(), DefaultLimit, 0)
	if !errors.Is(err, databaseError) {
		t.Fatalf("error = %v, want wrapped %v", err, databaseError)
	}
}

func TestListActiveFiltersByStatus(t *testing.T) {
	t.Parallel()

	exchangeID := uuid.New()
	exchanges := &fakeExchanges{
		items: []exchangemodel.Details{{ID: exchangeID, Status: "delivering"}},
		total: 2,
	}

	page, err := New(&fakeUsers{}, exchanges).
		ListActive(context.Background(), "delivering", 10, 5)
	if err != nil {
		t.Fatalf("ListActive() error = %v", err)
	}
	if exchanges.gotStatus != "delivering" || exchanges.gotLimit != 10 || exchanges.gotOffset != 5 {
		t.Fatalf("arguments = %q, %d, %d", exchanges.gotStatus, exchanges.gotLimit, exchanges.gotOffset)
	}
	if len(page.Exchanges) != 1 || page.Exchanges[0].ID != exchangeID || page.Total != 2 {
		t.Fatalf("page = %+v", page)
	}
}

func TestListActiveReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	page, err := New(&fakeUsers{}, &fakeExchanges{}).
		ListActive(context.Background(), "delivering", DefaultLimit, 0)
	if err != nil {
		t.Fatalf("ListActive() error = %v", err)
	}
	if page.Exchanges == nil {
		t.Fatal("Exchanges = nil, want []")
	}
}

func TestListActiveValidationStopsBeforeRepository(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status string
		limit  int32
		offset int32
	}{
		{name: "missing status", limit: DefaultLimit},
		{name: "unknown status", status: "cancelled", limit: DefaultLimit},
		{name: "zero limit", status: "delivering"},
		{name: "negative offset", status: "delivering", limit: DefaultLimit, offset: -1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exchanges := &fakeExchanges{}
			_, err := New(&fakeUsers{}, exchanges).
				ListActive(context.Background(), test.status, test.limit, test.offset)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("error = %v, want %v", err, ErrValidation)
			}
			if exchanges.listCalled {
				t.Fatal("repository called for invalid request")
			}
		})
	}
}
