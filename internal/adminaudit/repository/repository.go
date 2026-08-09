package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	adminauditmodel "github.com/sweetlife999/chain-of-trades-avito/internal/adminaudit/model"
	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

var (
	ErrNotFound = errors.New("user not found")
	ErrConflict = errors.New("user block state is unchanged")
)

type Repository struct{ queries *db.Queries }

func New(queries *db.Queries) *Repository { return &Repository{queries: queries} }

func (r *Repository) BlockUser(ctx context.Context, adminID, userID uuid.UUID) (adminauditmodel.UserBlockState, error) {
	row, err := r.queries.BlockUserForAdmin(ctx, db.BlockUserForAdminParams{AdminID: pgUUID(adminID), UserID: pgUUID(userID)})
	if errors.Is(err, pgx.ErrNoRows) {
		return adminauditmodel.UserBlockState{}, r.classifyUnchangedUser(ctx, userID)
	}
	if err != nil {
		return adminauditmodel.UserBlockState{}, fmt.Errorf("block user for admin: %w", err)
	}
	return stateFromFields(row.ID, row.Nickname, row.IsBlocked, row.BlockedAt, row.BlockedBy), nil
}

func (r *Repository) UnblockUser(ctx context.Context, adminID, userID uuid.UUID) (adminauditmodel.UserBlockState, error) {
	row, err := r.queries.UnblockUserForAdmin(ctx, db.UnblockUserForAdminParams{AdminID: pgUUID(adminID), UserID: pgUUID(userID)})
	if errors.Is(err, pgx.ErrNoRows) {
		return adminauditmodel.UserBlockState{}, r.classifyUnchangedUser(ctx, userID)
	}
	if err != nil {
		return adminauditmodel.UserBlockState{}, fmt.Errorf("unblock user for admin: %w", err)
	}
	return stateFromFields(row.ID, row.Nickname, row.IsBlocked, row.BlockedAt, row.BlockedBy), nil
}

func (r *Repository) classifyUnchangedUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.queries.GetUserByID(ctx, pgUUID(userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("classify unchanged user: %w", err)
	}
	return ErrConflict
}

func (r *Repository) List(ctx context.Context, filter adminauditmodel.Filter) ([]adminauditmodel.Entry, error) {
	rows, err := r.queries.ListAdminAuditLog(ctx, db.ListAdminAuditLogParams{
		AdminID: optionalUUID(filter.AdminID), Action: filter.Action,
		DateFrom: optionalTime(filter.From), DateTo: optionalTime(filter.To),
		PageLimit: filter.Limit, PageOffset: filter.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list admin audit log: %w", err)
	}
	entries := make([]adminauditmodel.Entry, len(rows))
	for index, row := range rows {
		entries[index] = adminauditmodel.Entry{
			ID: uuid.UUID(row.ID.Bytes), AdminID: uuid.UUID(row.AdminID.Bytes),
			Action: string(row.Action), TargetType: string(row.TargetType),
			TargetID: uuid.UUID(row.TargetID.Bytes), Metadata: row.Metadata,
			CreatedAt: row.CreatedAt.Time,
		}
	}
	return entries, nil
}

func (r *Repository) Count(ctx context.Context, filter adminauditmodel.Filter) (int64, error) {
	total, err := r.queries.CountAdminAuditLog(ctx, db.CountAdminAuditLogParams{
		AdminID: optionalUUID(filter.AdminID), Action: filter.Action,
		DateFrom: optionalTime(filter.From), DateTo: optionalTime(filter.To),
	})
	if err != nil {
		return 0, fmt.Errorf("count admin audit log: %w", err)
	}
	return total, nil
}

func stateFromFields(id pgtype.UUID, nickname string, blocked bool, at pgtype.Timestamptz, by pgtype.UUID) adminauditmodel.UserBlockState {
	state := adminauditmodel.UserBlockState{ID: uuid.UUID(id.Bytes), Nickname: nickname, IsBlocked: blocked}
	if at.Valid {
		value := at.Time
		state.BlockedAt = &value
	}
	if by.Valid {
		value := uuid.UUID(by.Bytes)
		state.BlockedBy = &value
	}
	return state
}

func pgUUID(id uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: [16]byte(id), Valid: true} }
func optionalUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgUUID(*id)
}
func optionalTime(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}
