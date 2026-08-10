package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	notificationmodel "github.com/sweetlife999/chain-of-trades-avito/internal/notification/model"
)

type fakeRepository struct {
	list          []notificationmodel.Notification
	unreadCount   int64
	markReadFound bool
	markAllCount  int64
	err           error
	lastFilter    notificationmodel.Filter
}

func (f *fakeRepository) List(
	_ context.Context,
	_ uuid.UUID,
	filter notificationmodel.Filter,
) ([]notificationmodel.Notification, error) {
	f.lastFilter = filter
	return f.list, f.err
}

func (f *fakeRepository) CountUnread(context.Context, uuid.UUID) (int64, error) {
	return f.unreadCount, f.err
}

func (f *fakeRepository) MarkRead(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return f.markReadFound, f.err
}

func (f *fakeRepository) MarkAllRead(context.Context, uuid.UUID) (int64, error) {
	return f.markAllCount, f.err
}

func TestListReturnsPageAndUnreadCount(t *testing.T) {
	t.Parallel()

	want := []notificationmodel.Notification{{ID: uuid.New(), Kind: "exchange_proposed"}}
	repository := &fakeRepository{list: want, unreadCount: 7}
	filter := notificationmodel.Filter{UnreadOnly: true, Limit: 25, Offset: 50}

	page, err := New(repository).List(context.Background(), uuid.New(), filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(page.Notifications) != 1 || page.Notifications[0].ID != want[0].ID {
		t.Fatalf("List() notifications = %+v, want %+v", page.Notifications, want)
	}
	if page.UnreadCount != 7 || page.Limit != 25 || page.Offset != 50 {
		t.Fatalf("List() page = %+v", page)
	}
	if repository.lastFilter != filter {
		t.Fatalf("List() filter = %+v, want %+v", repository.lastFilter, filter)
	}
}

func TestListValidatesPagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		userID uuid.UUID
		filter notificationmodel.Filter
	}{
		{name: "empty user", filter: notificationmodel.Filter{Limit: 50}},
		{name: "zero limit", userID: uuid.New()},
		{name: "too large limit", userID: uuid.New(), filter: notificationmodel.Filter{Limit: 101}},
		{name: "negative offset", userID: uuid.New(), filter: notificationmodel.Filter{Limit: 50, Offset: -1}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := New(&fakeRepository{}).List(context.Background(), test.userID, test.filter)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("List() error = %v, want %v", err, ErrValidation)
			}
		})
	}
}

func TestMarkReadRequiresOwnedNotification(t *testing.T) {
	t.Parallel()

	service := New(&fakeRepository{markReadFound: false})
	err := service.MarkRead(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("MarkRead() error = %v, want %v", err, ErrNotFound)
	}

	service = New(&fakeRepository{markReadFound: true})
	if err := service.MarkRead(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("MarkRead() error = %v", err)
	}
}

func TestMarkAllReadReturnsAffectedCount(t *testing.T) {
	t.Parallel()

	count, err := New(&fakeRepository{markAllCount: 4}).MarkAllRead(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("MarkAllRead() error = %v", err)
	}
	if count != 4 {
		t.Fatalf("MarkAllRead() count = %d, want 4", count)
	}
}

func TestRepositoryErrorsAreWrapped(t *testing.T) {
	t.Parallel()

	databaseErr := errors.New("database unavailable")
	service := New(&fakeRepository{err: databaseErr})
	_, err := service.List(
		context.Background(),
		uuid.New(),
		notificationmodel.Filter{Limit: 50},
	)
	if !errors.Is(err, databaseErr) {
		t.Fatalf("List() error = %v, want wrapped %v", err, databaseErr)
	}
}
