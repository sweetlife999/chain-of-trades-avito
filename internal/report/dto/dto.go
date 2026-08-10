package dto

import (
	"time"

	reportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/report/model"
)

type CreateReportRequest struct {
	MessageID string `json:"message_id" example:"3f2a1b4c-5d6e-4f70-8a91-b2c3d4e5f607"`
	Reason    string `json:"reason"     example:"abuse"`
	Comment   string `json:"comment"    example:"переходит на личности"`
}

type ReportResponse struct {
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	Reason    string    `json:"reason"`
	Comment   string    `json:"comment"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type ReportError struct {
	Error string `json:"error"`
}

func FromModel(report reportmodel.Report) ReportResponse {
	return ReportResponse{
		ID:        report.ID.String(),
		MessageID: report.MessageID.String(),
		Reason:    report.Reason,
		Comment:   report.Comment,
		Status:    report.Status,
		CreatedAt: report.CreatedAt,
	}
}
