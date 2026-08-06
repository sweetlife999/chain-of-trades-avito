package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	usermodel "github.com/sweetlife999/chain-of-trades-avito/internal/user/model"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrNicknameTaken = errors.New("nickname is already taken")
)

type Repository struct {
	queries *db.Queries
}

func New(queries *db.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Create(ctx context.Context, user usermodel.NewUser) (usermodel.User, error) {
	created, err := r.queries.CreateUser(ctx, db.CreateUserParams{
		Nickname:     user.Nickname,
		PasswordHash: user.PasswordHash,
		PhotoUrl:     optionalText(user.PhotoURL),
		Description:  user.Description,
	})
	if err != nil {
		return usermodel.User{}, fmt.Errorf("create user: %w", translateError(err))
	}

	return toModel(created)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (usermodel.User, error) {
	found, err := r.queries.GetUserByID(ctx, pgUUID(id))
	if err != nil {
		return usermodel.User{}, fmt.Errorf("get user by id: %w", translateError(err))
	}

	return toModel(found)
}

func (r *Repository) GetByNickname(ctx context.Context, nickname string) (usermodel.User, error) {
	found, err := r.queries.GetUserByNickname(ctx, nickname)
	if err != nil {
		return usermodel.User{}, fmt.Errorf("get user by nickname: %w", translateError(err))
	}

	return toModel(found)
}

func (r *Repository) IsAdmin(ctx context.Context, id uuid.UUID) (bool, error) {
	isAdmin, err := r.queries.IsUserAdmin(ctx, pgUUID(id))
	if err != nil {
		return false, fmt.Errorf("check user admin access: %w", err)
	}

	return isAdmin, nil
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, changes usermodel.Changes) (usermodel.User, error) {
	updated, err := r.queries.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		Nickname:    optionalText(changes.Nickname),
		PhotoUrl:    optionalText(changes.PhotoURL),
		Description: optionalText(changes.Description),
		ID:          pgUUID(id),
	})
	if err != nil {
		return usermodel.User{}, fmt.Errorf("update user: %w", translateError(err))
	}

	return toModel(updated)
}

func (r *Repository) Block(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	_, err := r.queries.AddUserBlockAndCancelProposals(ctx, db.AddUserBlockAndCancelProposalsParams{
		BlockerID: pgUUID(blockerID),
		BlockedID: pgUUID(blockedID),
	})
	if err != nil {
		return fmt.Errorf("add user block: %w", translateError(err))
	}

	return nil
}

func (r *Repository) ListBlocked(ctx context.Context, blockerID uuid.UUID) ([]usermodel.BlockedUser, error) {
	rows, err := r.queries.ListUserBlocks(ctx, pgUUID(blockerID))
	if err != nil {
		return nil, fmt.Errorf("list user blocks: %w", err)
	}

	blocked := make([]usermodel.BlockedUser, len(rows))
	for index, row := range rows {
		var photoURL *string
		if row.PhotoUrl.Valid {
			value := row.PhotoUrl.String
			photoURL = &value
		}

		blocked[index] = usermodel.BlockedUser{
			ID:        uuid.UUID(row.ID.Bytes),
			Nickname:  row.Nickname,
			PhotoURL:  photoURL,
			BlockedAt: row.BlockedAt.Time,
		}
	}

	return blocked, nil
}

func (r *Repository) Unblock(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	err := r.queries.DeleteUserBlock(ctx, db.DeleteUserBlockParams{
		BlockerID: pgUUID(blockerID),
		BlockedID: pgUUID(blockedID),
	})
	if err != nil {
		return fmt.Errorf("delete user block: %w", err)
	}

	return nil
}

func optionalText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *value, Valid: true}
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}

func toModel(user db.User) (usermodel.User, error) {
	var photoURL *string
	if user.PhotoUrl.Valid {
		value := user.PhotoUrl.String
		photoURL = &value
	}

	var rating *float64
	if user.Rating.Valid {
		value, err := user.Rating.Float64Value()
		if err != nil {
			return usermodel.User{}, fmt.Errorf("convert user rating: %w", err)
		}
		if value.Valid {
			ratingValue := value.Float64
			rating = &ratingValue
		}
	}

	return usermodel.User{
		ID:             uuid.UUID(user.ID.Bytes),
		Nickname:       user.Nickname,
		PasswordHash:   user.PasswordHash,
		PhotoURL:       photoURL,
		Description:    user.Description,
		DealsCompleted: user.DealsCompleted,
		DealsBroken:    user.DealsBroken,
		Rating:         rating,
		IsAdmin:        user.IsAdmin,
		CreatedAt:      user.CreatedAt.Time,
		UpdatedAt:      user.UpdatedAt.Time,
	}, nil
}

func translateError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == "user_blocks_blocked_id_fkey" {
		return ErrNotFound
	}
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_nickname_lower_key" {
		return ErrNicknameTaken
	}

	return err
}
