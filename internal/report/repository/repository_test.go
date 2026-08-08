package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestTranslateError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "not found", err: pgx.ErrNoRows, want: ErrNotFound},
		{
			name: "unique violation",
			err:  &pgconn.PgError{Code: "23505"},
			want: ErrDuplicate,
		},
		{
			name: "foreign key violation",
			err:  &pgconn.PgError{Code: "23503"},
			want: ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := translateError(test.err); !errors.Is(err, test.want) {
				t.Fatalf("translateError() = %v, want %v", err, test.want)
			}
		})
	}
}

func TestTranslateErrorPreservesUnknownError(t *testing.T) {
	t.Parallel()

	want := errors.New("database unavailable")
	if err := translateError(want); !errors.Is(err, want) {
		t.Fatalf("translateError() = %v, want original error", err)
	}
}
