package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	supportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/support/model"
)

var (
	ErrNotFound     = errors.New("support thread not found")
	ErrConflict     = errors.New("support thread state conflict")
	ErrActiveExists = errors.New("active support thread already exists")
)

const activeThreadConstraint = "support_threads_one_active_per_user_idx"

type Queries interface {
	CreateSupportThread(context.Context, db.CreateSupportThreadParams) (db.CreateSupportThreadRow, error)
	ListUserSupportThreads(context.Context, pgtype.UUID) ([]db.ListUserSupportThreadsRow, error)
	GetUserSupportThread(context.Context, db.GetUserSupportThreadParams) (db.GetUserSupportThreadRow, error)
	ListSupportMessages(context.Context, pgtype.UUID) ([]db.ListSupportMessagesRow, error)
	CreateUserSupportMessage(context.Context, db.CreateUserSupportMessageParams) (db.CreateUserSupportMessageRow, error)
	MarkSupportThreadReadByUser(context.Context, db.MarkSupportThreadReadByUserParams) (int64, error)
	CloseSupportThreadByUser(context.Context, db.CloseSupportThreadByUserParams) (int64, error)
	ListAdminSupportThreads(context.Context, db.ListAdminSupportThreadsParams) ([]db.ListAdminSupportThreadsRow, error)
	CountAdminSupportThreads(context.Context, db.CountAdminSupportThreadsParams) (int64, error)
	GetAdminSupportThread(context.Context, pgtype.UUID) (db.GetAdminSupportThreadRow, error)
	AssignSupportThread(context.Context, db.AssignSupportThreadParams) (int64, error)
	CreateAdminSupportMessage(context.Context, db.CreateAdminSupportMessageParams) (db.CreateAdminSupportMessageRow, error)
	CreateBotSupportMessage(context.Context, db.CreateBotSupportMessageParams) (db.CreateBotSupportMessageRow, error)
	SupportBotUserExists(context.Context, string) (bool, error)
	EscalateSupportThread(context.Context, pgtype.UUID) (int64, error)
	MarkSupportThreadReadByAdmin(context.Context, pgtype.UUID) (int64, error)
	CloseSupportThreadByAdmin(context.Context, db.CloseSupportThreadByAdminParams) (int64, error)
}

type Repository struct {
	queries Queries
}

func New(queries Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Create(
	ctx context.Context,
	userID uuid.UUID,
	subject string,
	body string,
) (supportmodel.Thread, error) {
	row, err := r.queries.CreateSupportThread(ctx, db.CreateSupportThreadParams{
		UserID: pgUUID(userID), Subject: subject, Body: body,
	})
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) &&
			postgresError.Code == "23505" && postgresError.ConstraintName == activeThreadConstraint {
			return supportmodel.Thread{}, ErrActiveExists
		}
		return supportmodel.Thread{}, fmt.Errorf("create support thread: %w", err)
	}

	return supportmodel.Thread{
		ID:        uuid.UUID(row.ID.Bytes),
		Subject:   row.Subject,
		Status:    string(row.Status),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func (r *Repository) ListUser(
	ctx context.Context,
	userID uuid.UUID,
) ([]supportmodel.Thread, error) {
	rows, err := r.queries.ListUserSupportThreads(ctx, pgUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("list user support threads: %w", err)
	}
	threads := make([]supportmodel.Thread, len(rows))
	for index, row := range rows {
		threads[index] = threadFromUserListRow(row)
	}
	return threads, nil
}

func (r *Repository) GetUser(
	ctx context.Context,
	threadID uuid.UUID,
	userID uuid.UUID,
) (supportmodel.Thread, error) {
	row, err := r.queries.GetUserSupportThread(ctx, db.GetUserSupportThreadParams{
		ThreadID: pgUUID(threadID), UserID: pgUUID(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return supportmodel.Thread{}, ErrNotFound
	}
	if err != nil {
		return supportmodel.Thread{}, fmt.Errorf("get user support thread: %w", err)
	}
	return threadFromUserRow(row), nil
}

func (r *Repository) ListMessages(
	ctx context.Context,
	threadID uuid.UUID,
) ([]supportmodel.Message, error) {
	rows, err := r.queries.ListSupportMessages(ctx, pgUUID(threadID))
	if err != nil {
		return nil, fmt.Errorf("list support messages: %w", err)
	}
	messages := make([]supportmodel.Message, len(rows))
	for index, row := range rows {
		messages[index] = messageFromListRow(row)
	}
	return messages, nil
}

func (r *Repository) CreateUserMessage(
	ctx context.Context,
	threadID uuid.UUID,
	userID uuid.UUID,
	body string,
) (supportmodel.Message, error) {
	row, err := r.queries.CreateUserSupportMessage(ctx, db.CreateUserSupportMessageParams{
		ThreadID: pgUUID(threadID), UserID: pgUUID(userID), Body: body,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return supportmodel.Message{}, ErrConflict
	}
	if err != nil {
		return supportmodel.Message{}, fmt.Errorf("create user support message: %w", err)
	}
	return messageFromUserCreateRow(row), nil
}

func (r *Repository) MarkReadByUser(
	ctx context.Context,
	threadID uuid.UUID,
	userID uuid.UUID,
) error {
	affected, err := r.queries.MarkSupportThreadReadByUser(ctx, db.MarkSupportThreadReadByUserParams{
		ThreadID: pgUUID(threadID), UserID: pgUUID(userID),
	})
	if err != nil {
		return fmt.Errorf("mark support thread read by user: %w", err)
	}
	if affected != 1 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) CloseByUser(
	ctx context.Context,
	threadID uuid.UUID,
	userID uuid.UUID,
) error {
	affected, err := r.queries.CloseSupportThreadByUser(ctx, db.CloseSupportThreadByUserParams{
		ThreadID: pgUUID(threadID), UserID: pgUUID(userID),
	})
	if err != nil {
		return fmt.Errorf("close support thread by user: %w", err)
	}
	if affected != 1 {
		return ErrConflict
	}
	return nil
}

func (r *Repository) ListAdmin(
	ctx context.Context,
	filter supportmodel.AdminFilter,
) ([]supportmodel.Thread, int64, error) {
	rows, err := r.queries.ListAdminSupportThreads(ctx, db.ListAdminSupportThreadsParams{
		StatusFilter: filter.Status, NeedsHuman: filter.NeedsHuman,
		PageLimit: filter.Limit, PageOffset: filter.Offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list admin support threads: %w", err)
	}
	total, err := r.queries.CountAdminSupportThreads(ctx, db.CountAdminSupportThreadsParams{
		StatusFilter: filter.Status, NeedsHuman: filter.NeedsHuman,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count admin support threads: %w", err)
	}
	threads := make([]supportmodel.Thread, len(rows))
	for index, row := range rows {
		threads[index] = threadFromAdminListRow(row)
	}
	return threads, total, nil
}

func (r *Repository) GetAdmin(
	ctx context.Context,
	threadID uuid.UUID,
) (supportmodel.Thread, error) {
	row, err := r.queries.GetAdminSupportThread(ctx, pgUUID(threadID))
	if errors.Is(err, pgx.ErrNoRows) {
		return supportmodel.Thread{}, ErrNotFound
	}
	if err != nil {
		return supportmodel.Thread{}, fmt.Errorf("get admin support thread: %w", err)
	}
	return threadFromAdminRow(row), nil
}

func (r *Repository) Assign(
	ctx context.Context,
	threadID uuid.UUID,
	adminID uuid.UUID,
) error {
	affected, err := r.queries.AssignSupportThread(ctx, db.AssignSupportThreadParams{
		ThreadID: pgUUID(threadID), AdminID: pgUUID(adminID),
	})
	if err != nil {
		return fmt.Errorf("assign support thread: %w", err)
	}
	if affected != 1 {
		return ErrConflict
	}
	return nil
}

func (r *Repository) CreateAdminMessage(
	ctx context.Context,
	threadID uuid.UUID,
	adminID uuid.UUID,
	body string,
) (supportmodel.Message, error) {
	row, err := r.queries.CreateAdminSupportMessage(ctx, db.CreateAdminSupportMessageParams{
		ThreadID: pgUUID(threadID), AdminID: pgUUID(adminID), Body: body,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return supportmodel.Message{}, ErrConflict
	}
	if err != nil {
		return supportmodel.Message{}, fmt.Errorf("create admin support message: %w", err)
	}
	return messageFromAdminCreateRow(row), nil
}

// BotUserExists отвечает, есть ли в базе служебный пользователь с этим ником. Нужен
// ровно для одного: ник в коде и ник в миграции обязаны совпадать, а расхождение иначе
// проявляется как молчание бота на каждом обращении — вставка не находит автора, отдаёт
// ноль строк, и это неотличимо от «обращение уже не ждёт ответа».
func (r *Repository) BotUserExists(ctx context.Context, botNickname string) (bool, error) {
	exists, err := r.queries.SupportBotUserExists(ctx, botNickname)
	if err != nil {
		return false, fmt.Errorf("check support bot user: %w", err)
	}
	return exists, nil
}

// Escalate помечает, что обращению нужен человек. Идемпотентно и намеренно не возвращает
// ErrConflict на ноль строк: ноль означает «уже эскалировано или закрыто», и для
// вызывающего это не ошибка, а отсутствие работы.
func (r *Repository) Escalate(ctx context.Context, threadID uuid.UUID) error {
	if _, err := r.queries.EscalateSupportThread(ctx, pgUUID(threadID)); err != nil {
		return fmt.Errorf("escalate support thread: %w", err)
	}
	return nil
}

// CreateBotMessage пишет ответ автоответчика. Автора запрос находит сам по нику, а
// ErrConflict здесь не ошибка выполнения, а нормальный исход: обращение успели закрыть,
// взять в работу или бот в нём уже отвечал. Вызывающий на такое только пишет в лог.
func (r *Repository) CreateBotMessage(
	ctx context.Context,
	threadID uuid.UUID,
	botNickname string,
	body string,
) (supportmodel.Message, error) {
	row, err := r.queries.CreateBotSupportMessage(ctx, db.CreateBotSupportMessageParams{
		ThreadID: pgUUID(threadID), BotNickname: botNickname, Body: body,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return supportmodel.Message{}, ErrConflict
	}
	if err != nil {
		return supportmodel.Message{}, fmt.Errorf("create bot support message: %w", err)
	}
	return messageFromBotCreateRow(row), nil
}

func (r *Repository) MarkReadByAdmin(ctx context.Context, threadID uuid.UUID) error {
	affected, err := r.queries.MarkSupportThreadReadByAdmin(ctx, pgUUID(threadID))
	if err != nil {
		return fmt.Errorf("mark support thread read by admin: %w", err)
	}
	if affected != 1 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) CloseByAdmin(
	ctx context.Context,
	threadID uuid.UUID,
	adminID uuid.UUID,
) error {
	affected, err := r.queries.CloseSupportThreadByAdmin(ctx, db.CloseSupportThreadByAdminParams{
		ThreadID: pgUUID(threadID), AdminID: pgUUID(adminID),
	})
	if err != nil {
		return fmt.Errorf("close support thread by admin: %w", err)
	}
	if affected != 1 {
		return ErrConflict
	}
	return nil
}

func threadFromUserListRow(row db.ListUserSupportThreadsRow) supportmodel.Thread {
	return supportmodel.Thread{
		ID: uuid.UUID(row.ID.Bytes), Subject: row.Subject, Status: string(row.Status),
		AssignedAdmin: optionalPerson(row.AssignedAdminID, row.AssignedAdminNickname, pgtype.Text{}, true),
		LastMessage:   row.LastMessageBody, LastMessageAt: optionalTime(row.LastMessageCreatedAt),
		UnreadCount: row.UnreadCount, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
		ClosedAt: optionalTime(row.ClosedAt),
	}
}

func threadFromUserRow(row db.GetUserSupportThreadRow) supportmodel.Thread {
	return supportmodel.Thread{
		ID: uuid.UUID(row.ID.Bytes), Subject: row.Subject, Status: string(row.Status),
		AssignedAdmin: optionalPerson(row.AssignedAdminID, row.AssignedAdminNickname, pgtype.Text{}, true),
		CreatedAt:     row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time, ClosedAt: optionalTime(row.ClosedAt),
	}
}

func threadFromAdminListRow(row db.ListAdminSupportThreadsRow) supportmodel.Thread {
	return supportmodel.Thread{
		ID: uuid.UUID(row.ID.Bytes), Subject: row.Subject, Status: string(row.Status),
		User:          &supportmodel.Person{ID: uuid.UUID(row.UserID.Bytes), Nickname: row.UserNickname, PhotoURL: optionalText(row.UserPhotoUrl)},
		AssignedAdmin: optionalPerson(row.AssignedAdminID, row.AssignedAdminNickname, pgtype.Text{}, true),
		LastMessage:   row.LastMessageBody, LastMessageAt: optionalTime(row.LastMessageCreatedAt),
		UnreadCount: row.UnreadCount, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
		ClosedAt: optionalTime(row.ClosedAt), EscalatedAt: optionalTime(row.EscalatedAt),
	}
}

func threadFromAdminRow(row db.GetAdminSupportThreadRow) supportmodel.Thread {
	return supportmodel.Thread{
		ID: uuid.UUID(row.ID.Bytes), Subject: row.Subject, Status: string(row.Status),
		User:          &supportmodel.Person{ID: uuid.UUID(row.UserID.Bytes), Nickname: row.UserNickname, PhotoURL: optionalText(row.UserPhotoUrl)},
		AssignedAdmin: optionalPerson(row.AssignedAdminID, row.AssignedAdminNickname, pgtype.Text{}, true),
		CreatedAt:     row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time, ClosedAt: optionalTime(row.ClosedAt),
		EscalatedAt: optionalTime(row.EscalatedAt),
	}
}

func messageFromListRow(row db.ListSupportMessagesRow) supportmodel.Message {
	return supportmodel.Message{
		ID: uuid.UUID(row.ID.Bytes), ThreadID: uuid.UUID(row.ThreadID.Bytes), Body: row.Body,
		Author:    supportmodel.Person{ID: uuid.UUID(row.AuthorID.Bytes), Nickname: row.AuthorNickname, PhotoURL: optionalText(row.AuthorPhotoUrl), IsAdmin: row.AuthorIsAdmin},
		CreatedAt: row.CreatedAt.Time,
	}
}

func messageFromUserCreateRow(row db.CreateUserSupportMessageRow) supportmodel.Message {
	return supportmodel.Message{
		ID: uuid.UUID(row.ID.Bytes), ThreadID: uuid.UUID(row.ThreadID.Bytes), Body: row.Body,
		Author:    supportmodel.Person{ID: uuid.UUID(row.AuthorID.Bytes), Nickname: row.AuthorNickname, PhotoURL: optionalText(row.AuthorPhotoUrl), IsAdmin: row.AuthorIsAdmin},
		CreatedAt: row.CreatedAt.Time,
	}
}

func messageFromAdminCreateRow(row db.CreateAdminSupportMessageRow) supportmodel.Message {
	return supportmodel.Message{
		ID: uuid.UUID(row.ID.Bytes), ThreadID: uuid.UUID(row.ThreadID.Bytes), Body: row.Body,
		Author:    supportmodel.Person{ID: uuid.UUID(row.AuthorID.Bytes), Nickname: row.AuthorNickname, PhotoURL: optionalText(row.AuthorPhotoUrl), IsAdmin: row.AuthorIsAdmin},
		CreatedAt: row.CreatedAt.Time,
	}
}

func messageFromBotCreateRow(row db.CreateBotSupportMessageRow) supportmodel.Message {
	return supportmodel.Message{
		ID: uuid.UUID(row.ID.Bytes), ThreadID: uuid.UUID(row.ThreadID.Bytes), Body: row.Body,
		Author:    supportmodel.Person{ID: uuid.UUID(row.AuthorID.Bytes), Nickname: row.AuthorNickname, PhotoURL: optionalText(row.AuthorPhotoUrl), IsAdmin: row.AuthorIsAdmin},
		CreatedAt: row.CreatedAt.Time,
	}
}

func optionalPerson(id pgtype.UUID, nickname pgtype.Text, photo pgtype.Text, isAdmin bool) *supportmodel.Person {
	if !id.Valid {
		return nil
	}
	return &supportmodel.Person{ID: uuid.UUID(id.Bytes), Nickname: nickname.String, PhotoURL: optionalText(photo), IsAdmin: isAdmin}
}

func optionalText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func optionalTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}
