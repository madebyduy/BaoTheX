package telegram

import (
	"context"
	"fmt"
	"strings"

	"repwire/internal/domain"
	"repwire/internal/postgres"
)

// MinDailyItems is the hard floor: below this we do not send at all — sending
// noise once makes a user mute the bot forever (spec section 16).
const MinDailyItems = 3

// Digest builds daily/weekly digest payloads.
type Digest struct {
	db      *postgres.DB
	baseURL string
}

// NewDigest constructs a Digest builder.
func NewDigest(db *postgres.DB, baseURL string) *Digest {
	return &Digest{db: db, baseURL: strings.TrimRight(baseURL, "/")}
}

// BuildDaily selects and formats a user's daily brief. It returns the formatted
// MarkdownV2 message, the chosen content ids, and ok=false when there are fewer
// than MinDailyItems candidates (caller must not send).
func (d *Digest) BuildDaily(ctx context.Context, userID int64, prefs *domain.NotificationPreferences) (msg string, ids []int64, ok bool, err error) {
	types := make([]string, len(prefs.ContentTypes))
	for i, t := range prefs.ContentTypes {
		types[i] = string(t)
	}
	// Over-fetch, then apply diversity (max 2/topic, max 2/source).
	candidates, err := d.db.Content.DailyCandidates(ctx, userID, types, prefs.HighlightsOnly, prefs.DailyMaxItems*4)
	if err != nil {
		return "", nil, false, err
	}
	chosen := d.diversify(ctx, candidates, prefs.DailyMaxItems)
	if len(chosen) < MinDailyItems {
		return "", nil, false, nil
	}

	ids = make([]int64, len(chosen))
	for i, c := range chosen {
		ids[i] = c.ID
	}
	return d.formatDaily(chosen), ids, true, nil
}

// BuildWeekly selects and formats the weekly research digest.
func (d *Digest) BuildWeekly(ctx context.Context, userID int64) (msg string, ids []int64, ok bool, err error) {
	items, err := d.db.Content.WeeklyResearchCandidates(ctx, userID, 5)
	if err != nil {
		return "", nil, false, err
	}
	if len(items) < 1 {
		return "", nil, false, nil
	}
	ids = make([]int64, len(items))
	for i, c := range items {
		ids[i] = c.ID
	}
	return d.formatWeekly(ctx, items), ids, true, nil
}

// diversify enforces at most 2 items per topic and per source.
func (d *Digest) diversify(ctx context.Context, items []domain.ContentItem, max int) []domain.ContentItem {
	perTopic := map[int64]int{}
	perSource := map[int64]int{}
	var out []domain.ContentItem
	for _, it := range items {
		if len(out) >= max {
			break
		}
		if perSource[it.SourceID] >= 2 {
			continue
		}
		topicID, _ := d.db.Content.PrimaryTopicID(ctx, it.ID)
		if topicID != 0 && perTopic[topicID] >= 2 {
			continue
		}
		out = append(out, it)
		perSource[it.SourceID]++
		if topicID != 0 {
			perTopic[topicID]++
		}
	}
	return out
}

// ---- Formatting (MarkdownV2) ----

func (d *Digest) formatDaily(items []domain.ContentItem) string {
	var b strings.Builder
	b.WriteString("🏋️ *RepWire Morning Brief*\n\n")
	for i, it := range items {
		b.WriteString(fmt.Sprintf("*%d\\. %s*\n", i+1, EscapeMarkdownV2(it.Title)))
		if it.Excerpt != nil && *it.Excerpt != "" {
			b.WriteString(EscapeMarkdownV2(clip(*it.Excerpt, 180)) + "\n")
		}
		b.WriteString(fmt.Sprintf("_%s_\n", EscapeMarkdownV2(it.SourceName)))
		b.WriteString(fmt.Sprintf("[%s](%s)\n\n", EscapeMarkdownV2(d.readLabel(it.Type)), d.itemURL(it)))
	}
	b.WriteString("⚙️ /settings · ⏸ /pause")
	return b.String()
}

func (d *Digest) formatWeekly(ctx context.Context, items []domain.ContentItem) string {
	var b strings.Builder
	b.WriteString("🔬 *RepWire Weekly Research*\n\n")
	for i, it := range items {
		b.WriteString(fmt.Sprintf("*%d\\. %s*\n", i+1, EscapeMarkdownV2(it.Title)))
		if rp, err := d.db.Content.GetResearch(ctx, it.ID); err == nil {
			if len(rp.Breakdown.Findings) > 0 {
				b.WriteString("• " + EscapeMarkdownV2(clip(rp.Breakdown.Findings[0], 160)) + "\n")
			}
			if rp.Breakdown.NotProven != nil && *rp.Breakdown.NotProven != "" {
				b.WriteString("⚠️ " + EscapeMarkdownV2(clip(*rp.Breakdown.NotProven, 160)) + "\n")
			}
		}
		b.WriteString(fmt.Sprintf("[Đọc breakdown](%s)\n\n", d.itemURL(it)))
	}
	b.WriteString("⚙️ /settings")
	return b.String()
}

func (d *Digest) readLabel(t domain.ContentType) string {
	switch t {
	case domain.ContentResearch:
		return "Đọc breakdown"
	case domain.ContentVideo:
		return "Xem"
	default:
		return "Đọc"
	}
}

// itemURL builds the canonical RepWire URL for an item by type.
func (d *Digest) itemURL(it domain.ContentItem) string {
	switch it.Type {
	case domain.ContentResearch:
		return fmt.Sprintf("%s/r/%d", d.baseURL, it.ID)
	case domain.ContentVideo:
		return fmt.Sprintf("%s/v/%d", d.baseURL, it.ID)
	default:
		return fmt.Sprintf("%s/c/%d", d.baseURL, it.ID)
	}
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n])) + "…"
}
