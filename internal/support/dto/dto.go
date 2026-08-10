package dto

import (
	"time"

	supportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/support/model"
)

type CreateThreadRequest struct {
	Subject string `json:"subject" example:"Не получается завершить обмен"`
	Body    string `json:"body" example:"Подскажите, что делать дальше?"`
}

type CreateMessageRequest struct {
	Body string `json:"body" example:"Спасибо, жду ответа"`
}

type PersonResponse struct {
	ID       string  `json:"id"`
	Nickname string  `json:"nickname"`
	PhotoURL *string `json:"photo_url"`
	IsAdmin  bool    `json:"is_admin"`
}

type ThreadResponse struct {
	ID            string          `json:"id"`
	User          *PersonResponse `json:"user,omitempty"`
	Subject       string          `json:"subject"`
	Status        string          `json:"status"`
	AssignedAdmin *PersonResponse `json:"assigned_admin"`
	LastMessage   string          `json:"last_message,omitempty"`
	LastMessageAt *time.Time      `json:"last_message_at"`
	UnreadCount   int64           `json:"unread_count"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	ClosedAt      *time.Time      `json:"closed_at"`
}

type MessageResponse struct {
	ID        string         `json:"id"`
	ThreadID  string         `json:"thread_id"`
	Author    PersonResponse `json:"author"`
	Body      string         `json:"body"`
	CreatedAt time.Time      `json:"created_at"`
}

type ThreadMessagesResponse struct {
	Thread   ThreadResponse    `json:"thread"`
	Messages []MessageResponse `json:"messages"`
}

type AdminPageResponse struct {
	Threads []ThreadResponse `json:"threads"`
	Total   int64            `json:"total"`
	Limit   int32            `json:"limit"`
	Offset  int32            `json:"offset"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func ThreadFromModel(thread supportmodel.Thread) ThreadResponse {
	return ThreadResponse{
		ID: thread.ID.String(), User: personFromModel(thread.User), Subject: thread.Subject,
		Status: thread.Status, AssignedAdmin: personFromModel(thread.AssignedAdmin),
		LastMessage: thread.LastMessage, LastMessageAt: thread.LastMessageAt,
		UnreadCount: thread.UnreadCount, CreatedAt: thread.CreatedAt,
		UpdatedAt: thread.UpdatedAt, ClosedAt: thread.ClosedAt,
	}
}

func ThreadsFromModels(threads []supportmodel.Thread) []ThreadResponse {
	result := make([]ThreadResponse, len(threads))
	for index, thread := range threads {
		result[index] = ThreadFromModel(thread)
	}
	return result
}

func MessageFromModel(message supportmodel.Message) MessageResponse {
	return MessageResponse{
		ID: message.ID.String(), ThreadID: message.ThreadID.String(),
		Author: *personFromModel(&message.Author), Body: message.Body, CreatedAt: message.CreatedAt,
	}
}

func MessagesFromModels(messages []supportmodel.Message) []MessageResponse {
	result := make([]MessageResponse, len(messages))
	for index, message := range messages {
		result[index] = MessageFromModel(message)
	}
	return result
}

func MessagesResponse(
	thread supportmodel.Thread,
	messages []supportmodel.Message,
) ThreadMessagesResponse {
	return ThreadMessagesResponse{
		Thread: ThreadFromModel(thread), Messages: MessagesFromModels(messages),
	}
}

func PageFromModel(page supportmodel.AdminPage) AdminPageResponse {
	return AdminPageResponse{
		Threads: ThreadsFromModels(page.Threads), Total: page.Total,
		Limit: page.Limit, Offset: page.Offset,
	}
}

func personFromModel(person *supportmodel.Person) *PersonResponse {
	if person == nil {
		return nil
	}
	return &PersonResponse{
		ID: person.ID.String(), Nickname: person.Nickname,
		PhotoURL: person.PhotoURL, IsAdmin: person.IsAdmin,
	}
}
