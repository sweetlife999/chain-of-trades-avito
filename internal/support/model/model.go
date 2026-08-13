package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusOpen       = "open"
	StatusInProgress = "in_progress"
	StatusClosed     = "closed"
)

type Person struct {
	ID       uuid.UUID
	Nickname string
	PhotoURL *string
	IsAdmin  bool
}

type Thread struct {
	ID            uuid.UUID
	User          *Person
	Subject       string
	Status        string
	AssignedAdmin *Person
	LastMessage   string
	LastMessageAt *time.Time
	UnreadCount   int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ClosedAt      *time.Time
	// EscalatedAt — момент, когда стало ясно, что автоответ не решил вопрос: пользователь
	// написал повторно или бот не справился с первого сообщения. Не статус: обращение
	// остаётся эскалированным и после того, как его взяли в работу.
	EscalatedAt *time.Time
}

type Message struct {
	ID        uuid.UUID
	ThreadID  uuid.UUID
	Author    Person
	Body      string
	CreatedAt time.Time
}

type AdminFilter struct {
	Status string
	// NeedsHuman оставляет в очереди только то, что ждёт модератора: эскалированные
	// обращения и те, на которые со стороны сервиса никто не ответил.
	NeedsHuman bool
	Limit      int32
	Offset     int32
}

type AdminPage struct {
	Threads []Thread
	Total   int64
	Limit   int32
	Offset  int32
}
