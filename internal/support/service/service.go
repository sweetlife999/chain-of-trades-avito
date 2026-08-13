package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	supportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/support/model"
	supportrepository "github.com/sweetlife999/chain-of-trades-avito/internal/support/repository"
)

const (
	minSubjectLength = 3
	maxSubjectLength = 160
	maxMessageLength = 2000
	DefaultLimit     = 20
	MaxLimit         = 100
)

var (
	ErrValidation   = errors.New("validation error")
	ErrNotFound     = supportrepository.ErrNotFound
	ErrConflict     = supportrepository.ErrConflict
	ErrActiveExists = supportrepository.ErrActiveExists
)

type ValidationError struct{ message string }

func (e *ValidationError) Error() string { return e.message }
func (e *ValidationError) Unwrap() error { return ErrValidation }

type Repository interface {
	Create(context.Context, uuid.UUID, string, string) (supportmodel.Thread, error)
	ListUser(context.Context, uuid.UUID) ([]supportmodel.Thread, error)
	GetUser(context.Context, uuid.UUID, uuid.UUID) (supportmodel.Thread, error)
	ListMessages(context.Context, uuid.UUID) ([]supportmodel.Message, error)
	CreateUserMessage(context.Context, uuid.UUID, uuid.UUID, string) (supportmodel.Message, error)
	MarkReadByUser(context.Context, uuid.UUID, uuid.UUID) error
	CloseByUser(context.Context, uuid.UUID, uuid.UUID) error
}

type Service struct {
	repository Repository
	// Нулевой bot — выключенная фича: пустой OLLAMA_URL не должен мешать поддержке
	// работать, поэтому проверок на nil у вызовов нет, они внутри Bot.
	bot *Bot
}

func New(repository Repository, bot *Bot) *Service {
	return &Service{repository: repository, bot: bot}
}

func (s *Service) Create(
	ctx context.Context,
	userID uuid.UUID,
	subject string,
	body string,
) (supportmodel.Thread, error) {
	if userID == uuid.Nil {
		return supportmodel.Thread{}, validationError("user id is required")
	}
	subject, err := validateSubject(subject)
	if err != nil {
		return supportmodel.Thread{}, err
	}
	body, err = validateMessage(body)
	if err != nil {
		return supportmodel.Thread{}, err
	}
	thread, err := s.repository.Create(ctx, userID, subject, body)
	if err != nil {
		return supportmodel.Thread{}, fmt.Errorf("create support thread: %w", err)
	}
	// Автоответ считается в фоне и только на первое сообщение: инференс на этом
	// сервере занимает то же ядро, что обслуживает HTTP, а ответ обращению нужен один.
	s.bot.Enqueue(thread.ID, subject, body)
	return thread, nil
}

func (s *Service) List(
	ctx context.Context,
	userID uuid.UUID,
) ([]supportmodel.Thread, error) {
	threads, err := s.repository.ListUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list support threads: %w", err)
	}
	if threads == nil {
		threads = []supportmodel.Thread{}
	}
	return threads, nil
}

func (s *Service) Messages(
	ctx context.Context,
	threadID uuid.UUID,
	userID uuid.UUID,
) (supportmodel.Thread, []supportmodel.Message, error) {
	thread, err := s.repository.GetUser(ctx, threadID, userID)
	if err != nil {
		return supportmodel.Thread{}, nil, fmt.Errorf("get support thread: %w", err)
	}
	messages, err := s.repository.ListMessages(ctx, threadID)
	if err != nil {
		return supportmodel.Thread{}, nil, fmt.Errorf("list support messages: %w", err)
	}
	if err := s.repository.MarkReadByUser(ctx, threadID, userID); err != nil {
		return supportmodel.Thread{}, nil, fmt.Errorf("mark support thread read: %w", err)
	}
	if messages == nil {
		messages = []supportmodel.Message{}
	}
	return thread, messages, nil
}

func (s *Service) Send(
	ctx context.Context,
	threadID uuid.UUID,
	userID uuid.UUID,
	body string,
) (supportmodel.Message, error) {
	body, err := validateMessage(body)
	if err != nil {
		return supportmodel.Message{}, err
	}
	thread, err := s.repository.GetUser(ctx, threadID, userID)
	if err != nil {
		return supportmodel.Message{}, fmt.Errorf("get support thread: %w", err)
	}
	if thread.Status == supportmodel.StatusClosed {
		return supportmodel.Message{}, ErrConflict
	}
	message, err := s.repository.CreateUserMessage(ctx, threadID, userID, body)
	if err != nil {
		return supportmodel.Message{}, fmt.Errorf("create support message: %w", err)
	}
	return message, nil
}

func (s *Service) Close(
	ctx context.Context,
	threadID uuid.UUID,
	userID uuid.UUID,
) (supportmodel.Thread, error) {
	thread, err := s.repository.GetUser(ctx, threadID, userID)
	if err != nil {
		return supportmodel.Thread{}, fmt.Errorf("get support thread: %w", err)
	}
	if thread.Status == supportmodel.StatusClosed {
		return thread, nil
	}
	if err := s.repository.CloseByUser(ctx, threadID, userID); err != nil {
		return supportmodel.Thread{}, fmt.Errorf("close support thread: %w", err)
	}
	return s.repository.GetUser(ctx, threadID, userID)
}

func validateSubject(value string) (string, error) {
	value = strings.TrimSpace(value)
	length := utf8.RuneCountInString(value)
	if length < minSubjectLength || length > maxSubjectLength {
		return "", validationError(fmt.Sprintf(
			"subject must be between %d and %d characters", minSubjectLength, maxSubjectLength,
		))
	}
	return value, nil
}

func validateMessage(value string) (string, error) {
	value = strings.TrimSpace(value)
	length := utf8.RuneCountInString(value)
	if length < 1 || length > maxMessageLength {
		return "", validationError(fmt.Sprintf(
			"message must be between 1 and %d characters", maxMessageLength,
		))
	}
	return value, nil
}

func validationError(message string) error { return &ValidationError{message: message} }
