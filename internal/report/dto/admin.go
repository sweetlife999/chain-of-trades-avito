package dto

import (
	"time"

	exchangedto "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/dto"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	reportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/report/model"
)

type AdminUserResponse struct {
	ID       string  `json:"id"`
	Nickname string  `json:"nickname"`
	PhotoURL *string `json:"photo_url"`
}

type AdminMessageResponse struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type AdminExchangeResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type AdminReportResponse struct {
	ID                string                `json:"id"`
	Reason            string                `json:"reason"`
	Comment           string                `json:"comment"`
	Status            string                `json:"status"`
	Reporter          AdminUserResponse     `json:"reporter"`
	Offender          AdminUserResponse     `json:"offender"`
	Message           AdminMessageResponse  `json:"message"`
	Exchange          AdminExchangeResponse `json:"exchange"`
	Assignee          *AdminUserResponse    `json:"assignee" extensions:"x-nullable"`
	CreatedAt         time.Time             `json:"created_at"`
	AssignedAt        *time.Time            `json:"assigned_at" extensions:"x-nullable"`
	ClosedAt          *time.Time            `json:"closed_at" extensions:"x-nullable"`
	ResolutionComment string                `json:"resolution_comment"`
}

type AdminDecisionRequest struct {
	Comment string `json:"comment" example:"Нарушение подтверждено"`
}

type PaginationResponse struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
	Total  int64 `json:"total"`
}

type AdminReportListResponse struct {
	Reports    []AdminReportResponse `json:"reports"`
	Pagination PaginationResponse    `json:"pagination"`
}

type AdminReportMessagesResponse struct {
	ReportID   string                        `json:"report_id"`
	ExchangeID string                        `json:"exchange_id"`
	Messages   []exchangedto.MessageResponse `json:"messages"`
}

func AdminReportFromModel(report reportmodel.AdminReport) AdminReportResponse {
	response := AdminReportResponse{
		ID:       report.ID.String(),
		Reason:   report.Reason,
		Comment:  report.Comment,
		Status:   report.Status,
		Reporter: adminUserFromModel(report.Reporter),
		Offender: adminUserFromModel(report.Offender),
		Message: AdminMessageResponse{
			ID:        report.Message.ID.String(),
			Body:      report.Message.Body,
			CreatedAt: report.Message.CreatedAt,
		},
		Exchange: AdminExchangeResponse{
			ID:     report.Exchange.ID.String(),
			Status: report.Exchange.Status,
		},
		CreatedAt:         report.CreatedAt,
		AssignedAt:        report.AssignedAt,
		ClosedAt:          report.ClosedAt,
		ResolutionComment: report.ResolutionComment,
	}

	if report.Assignee != nil {
		assignee := adminUserFromModel(*report.Assignee)
		response.Assignee = &assignee
	}

	return response
}

func AdminPageFromModel(page reportmodel.AdminPage) AdminReportListResponse {
	reports := make([]AdminReportResponse, len(page.Reports))
	for index, report := range page.Reports {
		reports[index] = AdminReportFromModel(report)
	}

	return AdminReportListResponse{
		Reports: reports,
		Pagination: PaginationResponse{
			Limit:  page.Limit,
			Offset: page.Offset,
			Total:  page.Total,
		},
	}
}

func AdminMessagesFromModels(
	report reportmodel.AdminReport,
	messages []exchangemodel.Message,
) AdminReportMessagesResponse {
	return AdminReportMessagesResponse{
		ReportID:   report.ID.String(),
		ExchangeID: report.Exchange.ID.String(),
		Messages:   exchangedto.MessagesFromModels(messages),
	}
}

func adminUserFromModel(user reportmodel.AdminUser) AdminUserResponse {
	return AdminUserResponse{
		ID:       user.ID.String(),
		Nickname: user.Nickname,
		PhotoURL: user.PhotoURL,
	}
}
