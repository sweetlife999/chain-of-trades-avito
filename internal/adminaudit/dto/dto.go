package dto

import (
	"encoding/json"
	"time"

	adminauditmodel "github.com/sweetlife999/chain-of-trades-avito/internal/adminaudit/model"
)

type UserBlockResponse struct {
	ID        string     `json:"id"`
	Nickname  string     `json:"nickname"`
	IsBlocked bool       `json:"is_blocked"`
	BlockedAt *time.Time `json:"blocked_at"`
	BlockedBy *string    `json:"blocked_by"`
}

type EntryResponse struct {
	ID         string          `json:"id"`
	AdminID    string          `json:"admin_id"`
	Action     string          `json:"action"`
	TargetType string          `json:"target_type"`
	TargetID   string          `json:"target_id"`
	Metadata   json.RawMessage `json:"metadata" swaggertype:"object"`
	CreatedAt  time.Time       `json:"created_at"`
}

type PageResponse struct {
	Entries []EntryResponse `json:"entries"`
	Limit   int32           `json:"limit"`
	Offset  int32           `json:"offset"`
	Total   int64           `json:"total"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func UserBlockFromModel(state adminauditmodel.UserBlockState) UserBlockResponse {
	var blockedBy *string
	if state.BlockedBy != nil {
		value := state.BlockedBy.String()
		blockedBy = &value
	}
	return UserBlockResponse{
		ID: state.ID.String(), Nickname: state.Nickname, IsBlocked: state.IsBlocked,
		BlockedAt: state.BlockedAt, BlockedBy: blockedBy,
	}
}

func PageFromModel(page adminauditmodel.Page) PageResponse {
	entries := make([]EntryResponse, len(page.Entries))
	for index, entry := range page.Entries {
		entries[index] = EntryResponse{
			ID: entry.ID.String(), AdminID: entry.AdminID.String(), Action: entry.Action,
			TargetType: entry.TargetType, TargetID: entry.TargetID.String(),
			Metadata: entry.Metadata, CreatedAt: entry.CreatedAt,
		}
	}
	return PageResponse{Entries: entries, Limit: page.Limit, Offset: page.Offset, Total: page.Total}
}
