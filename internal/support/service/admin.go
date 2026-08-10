package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	supportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/support/model"
)

var supportStatuses = []string{
	supportmodel.StatusOpen,
	supportmodel.StatusInProgress,
	supportmodel.StatusClosed,
}

type AdminRepository interface {
	ListAdmin(context.Context, supportmodel.AdminFilter) ([]supportmodel.Thread, int64, error)
	GetAdmin(context.Context, uuid.UUID) (supportmodel.Thread, error)
	ListMessages(context.Context, uuid.UUID) ([]supportmodel.Message, error)
	Assign(context.Context, uuid.UUID, uuid.UUID) error
	CreateAdminMessage(context.Context, uuid.UUID, uuid.UUID, string) (supportmodel.Message, error)
	MarkReadByAdmin(context.Context, uuid.UUID) error
	CloseByAdmin(context.Context, uuid.UUID, uuid.UUID) error
}

type AdminService struct{ repository AdminRepository }

func NewAdmin(repository AdminRepository) *AdminService {
	return &AdminService{repository: repository}
}

func (s *AdminService) List(
	ctx context.Context,
	filter supportmodel.AdminFilter,
) (supportmodel.AdminPage, error) {
	filter.Status = strings.TrimSpace(filter.Status)
	if filter.Status != "" && !slices.Contains(supportStatuses, filter.Status) {
		return supportmodel.AdminPage{}, validationError(
			"status must be one of: " + strings.Join(supportStatuses, ", "),
		)
	}
	if filter.Limit == 0 {
		filter.Limit = DefaultLimit
	}
	if filter.Limit < 1 || filter.Limit > MaxLimit {
		return supportmodel.AdminPage{}, validationError("limit must be between 1 and 100")
	}
	if filter.Offset < 0 {
		return supportmodel.AdminPage{}, validationError("offset must be non-negative")
	}
	threads, total, err := s.repository.ListAdmin(ctx, filter)
	if err != nil {
		return supportmodel.AdminPage{}, fmt.Errorf("list admin support threads: %w", err)
	}
	if threads == nil {
		threads = []supportmodel.Thread{}
	}
	return supportmodel.AdminPage{
		Threads: threads, Total: total, Limit: filter.Limit, Offset: filter.Offset,
	}, nil
}

func (s *AdminService) Messages(
	ctx context.Context,
	threadID uuid.UUID,
) (supportmodel.Thread, []supportmodel.Message, error) {
	thread, err := s.repository.GetAdmin(ctx, threadID)
	if err != nil {
		return supportmodel.Thread{}, nil, fmt.Errorf("get support thread: %w", err)
	}
	messages, err := s.repository.ListMessages(ctx, threadID)
	if err != nil {
		return supportmodel.Thread{}, nil, fmt.Errorf("list support messages: %w", err)
	}
	if err := s.repository.MarkReadByAdmin(ctx, threadID); err != nil {
		return supportmodel.Thread{}, nil, fmt.Errorf("mark support thread read: %w", err)
	}
	if messages == nil {
		messages = []supportmodel.Message{}
	}
	return thread, messages, nil
}

func (s *AdminService) Assign(
	ctx context.Context,
	threadID uuid.UUID,
	adminID uuid.UUID,
) (supportmodel.Thread, error) {
	thread, err := s.repository.GetAdmin(ctx, threadID)
	if err != nil {
		return supportmodel.Thread{}, fmt.Errorf("get support thread: %w", err)
	}
	if thread.Status == supportmodel.StatusClosed {
		return supportmodel.Thread{}, ErrConflict
	}
	if thread.AssignedAdmin != nil {
		if thread.AssignedAdmin.ID == adminID {
			return thread, nil
		}
		return supportmodel.Thread{}, ErrConflict
	}
	if err := s.repository.Assign(ctx, threadID, adminID); err != nil {
		return supportmodel.Thread{}, fmt.Errorf("assign support thread: %w", err)
	}
	return s.repository.GetAdmin(ctx, threadID)
}

func (s *AdminService) Send(
	ctx context.Context,
	threadID uuid.UUID,
	adminID uuid.UUID,
	body string,
) (supportmodel.Message, error) {
	body, err := validateMessage(body)
	if err != nil {
		return supportmodel.Message{}, err
	}
	thread, err := s.repository.GetAdmin(ctx, threadID)
	if err != nil {
		return supportmodel.Message{}, fmt.Errorf("get support thread: %w", err)
	}
	if thread.Status != supportmodel.StatusInProgress || thread.AssignedAdmin == nil ||
		thread.AssignedAdmin.ID != adminID {
		return supportmodel.Message{}, ErrConflict
	}
	message, err := s.repository.CreateAdminMessage(ctx, threadID, adminID, body)
	if err != nil {
		return supportmodel.Message{}, fmt.Errorf("create admin support message: %w", err)
	}
	return message, nil
}

func (s *AdminService) Close(
	ctx context.Context,
	threadID uuid.UUID,
	adminID uuid.UUID,
) (supportmodel.Thread, error) {
	thread, err := s.repository.GetAdmin(ctx, threadID)
	if err != nil {
		return supportmodel.Thread{}, fmt.Errorf("get support thread: %w", err)
	}
	if thread.Status == supportmodel.StatusClosed {
		return thread, nil
	}
	if thread.AssignedAdmin == nil || thread.AssignedAdmin.ID != adminID {
		return supportmodel.Thread{}, ErrConflict
	}
	if err := s.repository.CloseByAdmin(ctx, threadID, adminID); err != nil {
		return supportmodel.Thread{}, fmt.Errorf("close support thread: %w", err)
	}
	return s.repository.GetAdmin(ctx, threadID)
}
