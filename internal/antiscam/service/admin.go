package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	antiscammodel "github.com/sweetlife999/chain-of-trades-avito/internal/antiscam/model"
	antiscamrepository "github.com/sweetlife999/chain-of-trades-avito/internal/antiscam/repository"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

const (
	DefaultAdminLimit int32 = 20
	MaxAdminLimit     int32 = 100
	maxCommentLength        = 2000
)

var (
	ErrValidation      = errors.New("invalid antiscam input")
	ErrNotFound        = antiscamrepository.ErrNotFound
	ErrAlreadyReviewed = antiscamrepository.ErrAlreadyReviewed
)

type AdminRepository interface {
	List(context.Context, antiscammodel.Filter) ([]antiscammodel.Case, error)
	Count(context.Context, antiscammodel.Filter) (int64, error)
	Get(context.Context, uuid.UUID) (antiscammodel.Case, error)
	EvidenceIDs(context.Context, uuid.UUID) ([]uuid.UUID, error)
	Decide(context.Context, uuid.UUID, uuid.UUID, string, string) error
}

type MessageRepository interface {
	ListMessages(context.Context, uuid.UUID) ([]exchangemodel.Message, error)
}

type AdminService struct {
	repository AdminRepository
	messages   MessageRepository
}

func NewAdmin(repository AdminRepository, messages MessageRepository) *AdminService {
	return &AdminService{repository: repository, messages: messages}
}

func (s *AdminService) List(ctx context.Context, filter antiscammodel.Filter) (antiscammodel.Page, error) {
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Category = strings.TrimSpace(filter.Category)
	if filter.Limit == 0 {
		filter.Limit = DefaultAdminLimit
	}
	if filter.Limit < 1 || filter.Limit > MaxAdminLimit {
		return antiscammodel.Page{}, validation("limit must be between 1 and 100")
	}
	if filter.Offset < 0 {
		return antiscammodel.Page{}, validation("offset must be non-negative")
	}
	if filter.MinRisk < 0 || filter.MinRisk > 100 {
		return antiscammodel.Page{}, validation("min_risk must be between 0 and 100")
	}
	if filter.Status != "" && !slices.Contains([]string{antiscammodel.StatusOpen, antiscammodel.StatusResolved, antiscammodel.StatusDismissed}, filter.Status) {
		return antiscammodel.Page{}, validation("invalid status")
	}
	if filter.Category != "" && !slices.Contains(antiscammodel.Categories, filter.Category) {
		return antiscammodel.Page{}, validation("invalid category")
	}
	cases, err := s.repository.List(ctx, filter)
	if err != nil {
		return antiscammodel.Page{}, err
	}
	total, err := s.repository.Count(ctx, filter)
	if err != nil {
		return antiscammodel.Page{}, err
	}
	if cases == nil {
		cases = []antiscammodel.Case{}
	}
	return antiscammodel.Page{Cases: cases, Limit: filter.Limit, Offset: filter.Offset, Total: total}, nil
}

func (s *AdminService) Get(ctx context.Context, caseID uuid.UUID) (antiscammodel.Case, error) {
	if caseID == uuid.Nil {
		return antiscammodel.Case{}, validation("case id is required")
	}
	return s.repository.Get(ctx, caseID)
}

func (s *AdminService) Messages(ctx context.Context, caseID uuid.UUID) (antiscammodel.Case, []exchangemodel.Message, []uuid.UUID, error) {
	caseItem, err := s.Get(ctx, caseID)
	if err != nil {
		return antiscammodel.Case{}, nil, nil, err
	}
	messages, err := s.messages.ListMessages(ctx, caseItem.ExchangeID)
	if err != nil {
		return antiscammodel.Case{}, nil, nil, fmt.Errorf("list antiscam messages: %w", err)
	}
	evidence, err := s.repository.EvidenceIDs(ctx, caseID)
	if err != nil {
		return antiscammodel.Case{}, nil, nil, err
	}
	if messages == nil {
		messages = []exchangemodel.Message{}
	}
	return caseItem, messages, evidence, nil
}

func (s *AdminService) Decide(ctx context.Context, caseID, adminID uuid.UUID, decision, comment string) (antiscammodel.Case, error) {
	if caseID == uuid.Nil || adminID == uuid.Nil {
		return antiscammodel.Case{}, validation("case and admin ids are required")
	}
	if decision != antiscammodel.DecisionConfirmed && decision != antiscammodel.DecisionFalsePositive {
		return antiscammodel.Case{}, validation("invalid decision")
	}
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return antiscammodel.Case{}, validation("comment is required")
	}
	if utf8.RuneCountInString(comment) > maxCommentLength {
		return antiscammodel.Case{}, validation("comment is too long")
	}
	if err := s.repository.Decide(ctx, caseID, adminID, decision, comment); err != nil {
		return antiscammodel.Case{}, err
	}
	return s.Get(ctx, caseID)
}

func validation(message string) error { return fmt.Errorf("%w: %s", ErrValidation, message) }
