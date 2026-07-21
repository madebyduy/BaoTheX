package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"repwire/internal/domain"
	"repwire/internal/process"
)

// minAnalysisMaterials is how many independently-sourced, Vietnamese-readable
// pieces an analysis must stand on. Below three there is no cross-source
// picture to draw, only a rewrite of one outlet's copy.
const minAnalysisMaterials = 3

// maxClusterTranslations caps what one editorial decision may spend. Six
// translations plus two analysis calls is the worst case for a day's piece.
const maxClusterTranslations = 6

func (h *Handlers) handleGenerateAnalysis(ctx context.Context, j *domain.Job) error {
	var payload domain.AnalysisPayload
	if err := j.Unmarshal(&payload); err != nil {
		return err
	}
	if payload.ClusterID == 0 {
		return fmt.Errorf("cluster analysis: missing cluster id")
	}
	fail := func(err error) error {
		h.recordAnalysisFailure(ctx, j, payload.ClusterID, err)
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

	// Bring the cluster's own sources up to Vietnamese first. Ingest parks most
	// foreign articles untranslated to protect the budget, so without this step
	// a winning cluster would almost always fail the three-source bar below —
	// which is precisely why the analysis desk used to produce nothing.
	if err := h.translateClusterMaterials(ctx, payload.ClusterID); err != nil {
		return fail(err)
	}

	materials, err := h.DB.Analysis.GetMaterials(ctx, payload.ClusterID)
	if err != nil {
		return fail(err)
	}
	if len(materials) < minAnalysisMaterials {
		return fail(fmt.Errorf("cluster analysis: need %d publishable sources, have %d",
			minAnalysisMaterials, len(materials)))
	}
	claims, checkpointed, err := h.DB.Analysis.LoadClaimsCheckpoint(ctx, payload.ClusterID)
	if err != nil {
		return fail(err)
	}
	if !checkpointed {
		if err := h.DB.Analysis.UpdateProgress(ctx, payload.ClusterID, "extracting_claims", 0, 0, nil); err != nil {
			return fail(err)
		}
		claims, err = h.Summarizer.ExtractAnalysisClaims(ctx, materials[0].Title, materials)
		if err != nil {
			return fail(err)
		}
		if err := h.DB.Analysis.SaveClaimsCheckpoint(ctx, payload.ClusterID, *claims); err != nil {
			return fail(err)
		}
	}
	if err := h.DB.Analysis.UpdateProgress(ctx, payload.ClusterID, "writing_draft", 0, 0, nil); err != nil {
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
	h.Log.Info("cluster analysis draft ready for editorial review",
		"cluster", payload.ClusterID, "content", contentID, "materials", len(materials))
	return nil
}

// translateClusterMaterials translates the cluster's untranslated foreign
// articles on the editorial budget, which skips the hourly throughput cap that
// paces background wire work.
//
// A failure to translate one source is not fatal: the analysis only needs three
// readable materials, and refusing to write because a fourth outlet timed out
// would throw away the whole day's budget. Individual errors are logged and the
// three-source check below decides whether enough survived.
func (h *Handlers) translateClusterMaterials(ctx context.Context, clusterID int64) error {
	ids, err := h.DB.Content.IDsPendingTranslationForCluster(ctx, clusterID, maxClusterTranslations)
	if err != nil {
		return err
	}
	if err := h.DB.Analysis.UpdateProgress(ctx, clusterID, "translating", 0, len(ids), nil); err != nil {
		return err
	}
	for i, id := range ids {
		if ok, err := h.Summarizer.EditorialBudgetOK(ctx); err != nil {
			return err
		} else if !ok {
			h.Log.Warn("stopped translating cluster materials: daily budget reached",
				"cluster", clusterID)
			return nil
		}
		if err := h.translateOneMaterial(ctx, id); err != nil {
			h.Log.Warn("cluster material translation failed",
				"cluster", clusterID, "content", id, "err", err)
		}
		if err := h.DB.Analysis.UpdateProgress(ctx, clusterID, "translating", i+1, len(ids), nil); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handlers) recordAnalysisFailure(ctx context.Context, j *domain.Job, clusterID int64, cause error) {
	stateCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if j.Attempts >= j.MaxAttempts {
		_ = h.DB.Analysis.MarkFailed(stateCtx, clusterID, cause)
		return
	}
	next := time.Now().Add(retryDelay(j.Attempts, cause))
	stage := "retrying"
	if errors.Is(cause, process.ErrDailyQuotaExceeded) {
		stage = "waiting_quota"
	}
	_ = h.DB.Analysis.UpdateProgress(stateCtx, clusterID, stage, 0, 0, &next)
}

func (h *Handlers) translateOneMaterial(ctx context.Context, contentID int64) error {
	body, err := h.DB.Content.GetBody(ctx, contentID)
	if err != nil {
		return err
	}
	item, err := h.DB.Content.Get(ctx, contentID)
	if err != nil {
		return err
	}
	out, err := h.Summarizer.TranslateAndSummarizeEditorial(ctx, item.Title, body.OriginalBody)
	if err != nil {
		return err
	}
	if !looksVietnamese(out.VietnameseTitle) || !looksVietnamese(out.VietnameseBody) ||
		strings.HasPrefix(strings.TrimSpace(out.VietnameseBody), "{") {
		return fmt.Errorf("translation did not return Vietnamese prose")
	}
	if err := h.DB.Content.SetVietnameseContent(ctx, contentID, out.VietnameseTitle, out.VietnameseBody); err != nil {
		return err
	}
	// Re-cluster on the Vietnamese title now that one exists: the trigram match
	// runs on COALESCE(translated_title, title), so translating a member can pull
	// it together with Vietnamese coverage of the same event.
	if err := h.DB.Content.ClusterContent(ctx, contentID, out.VietnameseTitle); err != nil {
		return err
	}
	return h.DB.Content.SetSummary(ctx, contentID, out.Summary, out.KeyPoints, domain.StatusReady)
}
