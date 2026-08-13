package repository

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	antiscammodel "github.com/sweetlife999/chain-of-trades-avito/internal/antiscam/model"
	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

var (
	ErrNotFound        = errors.New("antiscam case not found")
	ErrAlreadyReviewed = errors.New("antiscam case is already reviewed")
)

type Repository struct{ queries *db.Queries }

func New(queries *db.Queries) *Repository { return &Repository{queries: queries} }

func (r *Repository) Claim(ctx context.Context) (antiscammodel.Job, error) {
	row, err := r.queries.ClaimAntiscamAnalysis(ctx)
	if err != nil {
		return antiscammodel.Job{}, err
	}
	return antiscammodel.Job{ID: uuid.UUID(row.ID.Bytes), MessageID: uuid.UUID(row.MessageID.Bytes), Attempts: row.Attempts}, nil
}

func (r *Repository) Input(ctx context.Context, messageID uuid.UUID) (antiscammodel.Message, error) {
	row, err := r.queries.GetAntiscamInput(ctx, pgUUID(messageID))
	if err != nil {
		return antiscammodel.Message{}, fmt.Errorf("get antiscam input: %w", err)
	}
	return antiscammodel.Message{ID: uuid.UUID(row.MessageID.Bytes), ExchangeID: uuid.UUID(row.ChainID.Bytes), AuthorID: uuid.UUID(row.AuthorID.Bytes), AuthorNickname: row.AuthorNickname, Body: row.Body.String}, nil
}

func (r *Repository) Context(ctx context.Context, messageID uuid.UUID) ([]antiscammodel.ContextMessage, error) {
	rows, err := r.queries.ListAntiscamContext(ctx, pgUUID(messageID))
	if err != nil {
		return nil, fmt.Errorf("list antiscam context: %w", err)
	}
	messages := make([]antiscammodel.ContextMessage, len(rows))
	for index, row := range rows {
		messages[len(rows)-1-index] = antiscammodel.ContextMessage{ID: uuid.UUID(row.ID.Bytes), AuthorID: uuid.UUID(row.AuthorID.Bytes), Nickname: row.Nickname, Body: row.Body.String, CreatedAt: row.CreatedAt.Time}
	}
	return messages, nil
}

func (r *Repository) Complete(ctx context.Context, jobID uuid.UUID, analysis antiscammodel.Analysis) error {
	var category db.NullAntiscamCategory
	if analysis.Category != nil {
		category = db.NullAntiscamCategory{AntiscamCategory: db.AntiscamCategory(*analysis.Category), Valid: true}
	}
	var modelSuspicious pgtype.Bool
	if analysis.ModelSuspicious != nil {
		modelSuspicious = pgtype.Bool{Bool: *analysis.ModelSuspicious, Valid: true}
	}
	var modelSeverity pgtype.Text
	if analysis.ModelSeverity != nil {
		modelSeverity = pgtype.Text{String: *analysis.ModelSeverity, Valid: true}
	}
	return r.queries.CompleteAntiscamAnalysis(ctx, db.CompleteAntiscamAnalysisParams{
		RuleScore: analysis.RuleScore, RuleHits: analysis.RuleHits, ModelSuspicious: modelSuspicious,
		ModelSeverity: modelSeverity, Category: category, Reason: analysis.Reason, Evidence: analysis.Evidence,
		Risk: analysis.Risk, ModelName: analysis.ModelName, PromptVersion: analysis.PromptVersion,
		AnalysisID: pgUUID(jobID), IsSuspicious: analysis.Suspicious,
	})
}

func (r *Repository) Retry(ctx context.Context, jobID uuid.UUID, delaySeconds int32, cause error) error {
	return r.queries.RetryAntiscamAnalysis(ctx, db.RetryAntiscamAnalysisParams{RetrySeconds: delaySeconds, LastError: cause.Error(), AnalysisID: pgUUID(jobID)})
}

func (r *Repository) List(ctx context.Context, filter antiscammodel.Filter) ([]antiscammodel.Case, error) {
	rows, err := r.queries.ListAntiscamCasesForAdmin(ctx, db.ListAntiscamCasesForAdminParams{CaseStatus: filter.Status, CaseCategory: filter.Category, MinRisk: filter.MinRisk, PageLimit: filter.Limit, PageOffset: filter.Offset})
	if err != nil {
		return nil, fmt.Errorf("list antiscam cases: %w", err)
	}
	cases := make([]antiscammodel.Case, len(rows))
	for index, row := range rows {
		cases[index] = caseFromFields(row.ID, row.ChainID, row.SuspectUserID, row.Status, row.Risk, row.Category, row.Reason, row.ReviewedBy, row.Decision, row.ResolutionComment, row.CreatedAt, row.UpdatedAt, row.ClosedAt, row.SuspectNickname, row.SuspectPhotoUrl, row.ReviewerNickname, row.LatestMessageID, row.LatestMessageBody, row.LatestMessageCreatedAt, row.EvidenceCount)
	}
	return cases, nil
}

func (r *Repository) Count(ctx context.Context, filter antiscammodel.Filter) (int64, error) {
	return r.queries.CountAntiscamCasesForAdmin(ctx, db.CountAntiscamCasesForAdminParams{CaseStatus: filter.Status, CaseCategory: filter.Category, MinRisk: filter.MinRisk})
}

func (r *Repository) Get(ctx context.Context, caseID uuid.UUID) (antiscammodel.Case, error) {
	row, err := r.queries.GetAntiscamCaseForAdmin(ctx, pgUUID(caseID))
	if errors.Is(err, pgx.ErrNoRows) {
		return antiscammodel.Case{}, ErrNotFound
	}
	if err != nil {
		return antiscammodel.Case{}, fmt.Errorf("get antiscam case: %w", err)
	}
	return caseFromFields(row.ID, row.ChainID, row.SuspectUserID, row.Status, row.Risk, row.Category, row.Reason, row.ReviewedBy, row.Decision, row.ResolutionComment, row.CreatedAt, row.UpdatedAt, row.ClosedAt, row.SuspectNickname, row.SuspectPhotoUrl, row.ReviewerNickname, row.LatestMessageID, row.LatestMessageBody, row.LatestMessageCreatedAt, row.EvidenceCount), nil
}

func (r *Repository) EvidenceIDs(ctx context.Context, caseID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.queries.ListAntiscamEvidenceIDs(ctx, pgUUID(caseID))
	if err != nil {
		return nil, fmt.Errorf("list antiscam evidence: %w", err)
	}
	ids := make([]uuid.UUID, len(rows))
	for index, row := range rows {
		ids[index] = uuid.UUID(row.Bytes)
	}
	return ids, nil
}

func (r *Repository) Decide(ctx context.Context, caseID, adminID uuid.UUID, decision, comment string) error {
	affected, err := r.queries.DecideAntiscamCaseForAdmin(ctx, db.DecideAntiscamCaseForAdminParams{Decision: db.NullAntiscamDecision{AntiscamDecision: db.AntiscamDecision(decision), Valid: true}, AdminID: pgUUID(adminID), ResolutionComment: comment, CaseID: pgUUID(caseID)})
	if err != nil {
		return fmt.Errorf("decide antiscam case: %w", err)
	}
	if affected == 1 {
		return nil
	}
	if _, err := r.Get(ctx, caseID); err != nil {
		return err
	}
	return ErrAlreadyReviewed
}

func caseFromFields(id, chainID, suspectID pgtype.UUID, status db.AntiscamCaseStatus, risk int32, category db.AntiscamCategory, reason string, reviewedBy pgtype.UUID, decision db.NullAntiscamDecision, comment string, createdAt, updatedAt, closedAt pgtype.Timestamptz, suspectNickname string, suspectPhoto, reviewerNickname pgtype.Text, latestID pgtype.UUID, latestBody pgtype.Text, latestCreatedAt pgtype.Timestamptz, evidenceCount int64) antiscammodel.Case {
	result := antiscammodel.Case{ID: uuid.UUID(id.Bytes), ExchangeID: uuid.UUID(chainID.Bytes), Status: string(status), Risk: risk, Category: string(category), Reason: reason, ResolutionComment: comment, EvidenceCount: evidenceCount, Suspect: antiscammodel.User{ID: uuid.UUID(suspectID.Bytes), Nickname: suspectNickname, PhotoURL: optionalText(suspectPhoto)}, LatestEvidence: antiscammodel.Evidence{ID: uuid.UUID(latestID.Bytes), Body: latestBody.String, CreatedAt: latestCreatedAt.Time}, CreatedAt: createdAt.Time, UpdatedAt: updatedAt.Time}
	if reviewedBy.Valid {
		result.Reviewer = &antiscammodel.User{ID: uuid.UUID(reviewedBy.Bytes), Nickname: reviewerNickname.String}
	}
	if decision.Valid {
		value := string(decision.AntiscamDecision)
		result.Decision = &value
	}
	if closedAt.Valid {
		value := closedAt.Time
		result.ClosedAt = &value
	}
	return result
}

func pgUUID(value uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: value, Valid: true} }
func optionalText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func ValidCategory(value string) bool { return slices.Contains(antiscammodel.Categories, value) }
