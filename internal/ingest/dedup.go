package ingest

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"repwire/internal/domain"
	"repwire/internal/postgres"
)

// Store normalises a raw item, applies the three dedup tiers and inserts the
// content_items row plus its subtype row. It returns the new content id, or 0
// if the item was a duplicate and skipped.
//
// Tier 1 (hard): url_hash UNIQUE.
// Tier 2 (identifier): doi / pmid / youtube_id / episode_guid.
// Tier 3 (soft, title similarity) is handled separately in ProcessSoftDedup so
// it can run after insert with the item's own id excluded.
func Store(ctx context.Context, db *postgres.DB, src *domain.Source, raw RawItem) (int64, error) {
	canonical, err := Normalize(raw.URL)
	if err != nil {
		return 0, err
	}
	urlHash := URLHash(canonical)

	// Tier 1: hard dedup on url_hash.
	if exists, err := db.Content.ExistsByURLHash(ctx, urlHash); err != nil {
		return 0, err
	} else if exists {
		if raw.Body != nil && strings.TrimSpace(*raw.Body) != "" {
			lang := raw.Language
			if lang == "" {
				lang = src.DefaultLang
			}
			if err := db.Content.UpsertBodyByURLHash(ctx, urlHash, lang, *raw.Body); err != nil {
				return 0, err
			}
		}
		return 0, nil
	}

	// Tier 2: identifier dedup by subtype.
	switch raw.Type {
	case domain.ContentResearch:
		if raw.Research != nil {
			if exists, err := db.Content.ExistsByDOIOrPMID(ctx, raw.Research.DOI, raw.Research.PMID); err != nil {
				return 0, err
			} else if exists {
				return 0, nil
			}
		}
	case domain.ContentVideo:
		if raw.Video != nil {
			if exists, err := db.Content.ExistsByYouTubeID(ctx, raw.Video.YouTubeID); err != nil {
				return 0, err
			} else if exists {
				return 0, nil
			}
		}
	case domain.ContentPodcast:
		if raw.Podcast != nil {
			if exists, err := db.Content.ExistsByEpisodeGUID(ctx, raw.Podcast.EpisodeGUID); err != nil {
				return 0, err
			} else if exists {
				return 0, nil
			}
		}
	}

	lang := raw.Language
	if lang == "" {
		lang = src.DefaultLang
	}
	titleHash := NormalizeTitle(raw.Title)

	item := &domain.ContentItem{
		SourceID:     src.ID,
		Type:         raw.Type,
		Status:       domain.StatusDiscovered,
		Title:        raw.Title,
		CanonicalURL: canonical,
		URLHash:      urlHash,
		TitleHash:    &titleHash,
		ImageURL:     raw.ImageURL,
		Excerpt:      raw.Excerpt,
		KeyPoints:    []string{},
		Language:     lang,
		PublishedAt:  raw.Published,
	}

	var newID int64
	err = db.WithTx(ctx, func(tx pgx.Tx) error {
		id, err := db.Content.InsertItem(ctx, tx, item)
		if err != nil {
			return err
		}
		if id == 0 {
			return nil // lost a race on url_hash
		}
		newID = id
		if raw.Body != nil && strings.TrimSpace(*raw.Body) != "" {
			if err := db.Content.InsertBody(ctx, tx, id, lang, *raw.Body); err != nil {
				return err
			}
		}
		switch raw.Type {
		case domain.ContentArticle, domain.ContentAnnouncement, domain.ContentEvent:
			a := raw.Article
			if a == nil {
				a = &domain.Article{}
			}
			a.ContentID = id
			return db.Content.InsertArticle(ctx, tx, a)
		case domain.ContentResearch:
			if raw.Research == nil {
				return nil
			}
			raw.Research.ContentID = id
			return db.Content.InsertResearch(ctx, tx, raw.Research)
		case domain.ContentVideo:
			if raw.Video == nil {
				return nil
			}
			raw.Video.ContentID = id
			return db.Content.InsertVideo(ctx, tx, raw.Video)
		case domain.ContentPodcast:
			if raw.Podcast == nil {
				return nil
			}
			raw.Podcast.ContentID = id
			return db.Content.InsertPodcast(ctx, tx, raw.Podcast)
		}
		return nil
	})
	return newID, err
}
