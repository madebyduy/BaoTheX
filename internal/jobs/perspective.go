package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"repwire/internal/domain"
)

// handleGeneratePerspective drafts a Góc nhìn from a single article on editor
// demand. It is the single-source counterpart to handleGenerateAnalysis: no
// cross-source claim extraction, just one evaluative pass over the article's own
// text. The draft is written for review and never published by this job.
func (h *Handlers) handleGeneratePerspective(ctx context.Context, j *domain.Job) error {
	var payload domain.PerspectivePayload
	if err := j.Unmarshal(&payload); err != nil {
		return err
	}
	if payload.ContentID == 0 {
		return fmt.Errorf("article perspective: missing content id")
	}
	fail := func(err error) error {
		if payload.ClusterID != 0 {
			h.recordAnalysisFailure(ctx, j, payload.ClusterID, err)
		}
		return err
	}
	if h.Summarizer == nil || !h.Summarizer.Enabled() {
		return fail(fmt.Errorf("article perspective: LLM is not configured"))
	}
	if ok, err := h.Summarizer.EditorialBudgetOK(ctx); err != nil {
		return fail(err)
	} else if !ok {
		return fail(fmt.Errorf("article perspective: daily LLM budget exceeded"))
	}

	item, err := h.DB.Content.Get(ctx, payload.ContentID)
	if err != nil {
		return fail(err)
	}
	body, err := h.DB.Content.GetBody(ctx, payload.ContentID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return fail(err)
	}

	// Prefer the Vietnamese body when it is ready; otherwise fall back to the
	// original text, and finally to the summary — a perspective needs prose to
	// stand on, but it should not refuse just because a native piece was stored
	// without a separate translated body.
	basis := ""
	if body != nil {
		if body.VietnameseBody != nil && strings.EqualFold(body.TranslationStatus, "ready") &&
			strings.TrimSpace(*body.VietnameseBody) != "" {
			basis = *body.VietnameseBody
		} else {
			basis = body.OriginalBody
		}
	}
	summary := ""
	if item.Summary != nil {
		summary = *item.Summary
	}
	if strings.TrimSpace(basis) == "" {
		basis = summary
	}
	if strings.TrimSpace(basis) == "" {
		return fail(fmt.Errorf("article perspective: no readable text for content %d", payload.ContentID))
	}

	if err := h.DB.Analysis.UpdateProgress(ctx, payload.ClusterID, "writing_draft", 0, 0, nil); err != nil {
		return fail(err)
	}
	draft, err := h.Summarizer.WriteArticlePerspective(ctx, item.Title, item.SourceName, basis, summary)
	if err != nil {
		return fail(err)
	}
	// Reuse the cluster-analysis draft path so the perspective lands in the same
	// review-and-publish desk. A single-article perspective carries no
	// cross-source claim map, so the claim arrays are empty.
	draftContentID, err := h.DB.Analysis.CreateDraft(ctx, payload.ClusterID, domain.AnalysisClaims{
		Consensus:     []string{},
		Conflicts:     []string{},
		UniqueClaims:  []string{},
		OpenQuestions: []string{},
	}, *draft)
	if err != nil {
		return fail(err)
	}
	h.Log.Info("article perspective draft ready for editorial review",
		"source_content", payload.ContentID, "cluster", payload.ClusterID, "draft_content", draftContentID)
	return nil
}
