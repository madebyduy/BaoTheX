package jobs

import (
	"context"
	"fmt"

	"repwire/internal/domain"
)

func (h *Handlers) handleGenerateAnalysis(ctx context.Context, j *domain.Job) error {
	var payload domain.AnalysisPayload
	if err := j.Unmarshal(&payload); err != nil {
		return err
	}
	if payload.ClusterID == 0 {
		return fmt.Errorf("cluster analysis: missing cluster id")
	}
	fail := func(err error) error {
		_ = h.DB.Analysis.MarkFailed(ctx, payload.ClusterID, err)
		return err
	}
	if h.Summarizer == nil || !h.Summarizer.Enabled() {
		return fail(fmt.Errorf("cluster analysis: LLM is not configured"))
	}
	if ok, err := h.Summarizer.EditorialBudgetOK(ctx); err != nil {
		return fail(err)
	} else if !ok {
		return fail(fmt.Errorf("cluster analysis: daily LLM budget exceeded"))
	}
	materials, err := h.DB.Analysis.GetMaterials(ctx, payload.ClusterID)
	if err != nil {
		return fail(err)
	}
	if len(materials) < 3 {
		return fail(fmt.Errorf("cluster analysis: requires at least three publishable sources"))
	}
	claims, err := h.Summarizer.ExtractAnalysisClaims(ctx, materials[0].Title, materials)
	if err != nil {
		return fail(err)
	}
	draft, err := h.Summarizer.WriteClusterAnalysis(ctx, materials[0].Title, materials, *claims)
	if err != nil {
		return fail(err)
	}
	contentID, err := h.DB.Analysis.CreateDraft(ctx, payload.ClusterID, *claims, *draft)
	if err != nil {
		return fail(err)
	}
	h.Log.Info("cluster analysis draft ready for editorial review", "cluster", payload.ClusterID, "content", contentID)
	return nil
}
