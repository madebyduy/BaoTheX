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

// Digest builds the Telegram payloads: the scheduled daily brief and the
// news-driven follow alert.
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

// MaxFollowAlertItems caps one alert. Past a handful it stops being an alert
// and becomes an unscheduled digest, which is the thing the daily brief exists
// to be — and which the reader chose the hour for.
const MaxFollowAlertItems = 3

// BuildFollowAlert formats a nudge about new stories in the topics a user
// follows. ok=false means there is nothing worth interrupting them for, which is
// the common case and not an error.
func (d *Digest) BuildFollowAlert(ctx context.Context, userID int64, prefs *domain.NotificationPreferences) (msg string, ids []int64, ok bool, err error) {
	candidates, err := d.db.Content.FollowAlertCandidates(ctx, userID, prefs.HighlightsOnly, MaxFollowAlertItems*3)
	if err != nil {
		return "", nil, false, err
	}
	chosen := d.diversify(ctx, candidates, MaxFollowAlertItems)
	if len(chosen) == 0 {
		return "", nil, false, nil
	}
	ids = make([]int64, len(chosen))
	for i, c := range chosen {
		ids[i] = c.ID
	}
	return d.formatFollowAlert(chosen), ids, true, nil
}

func (d *Digest) formatFollowAlert(items []domain.ContentItem) string {
	var b strings.Builder
	if len(items) == 1 {
		b.WriteString("🔔 *Chủ đề bạn theo dõi vừa có tin mới*\n\n")
	} else {
		b.WriteString(fmt.Sprintf("🔔 *%d tin mới trong chủ đề bạn theo dõi*\n\n", len(items)))
	}
	for _, it := range items {
		b.WriteString(fmt.Sprintf("*%s*\n", EscapeMarkdownV2(it.Title)))
		if it.Excerpt != nil && *it.Excerpt != "" {
			b.WriteString(EscapeMarkdownV2(clip(*it.Excerpt, 160)) + "\n")
		}
		b.WriteString(fmt.Sprintf("_%s_\n", EscapeMarkdownV2(it.SourceName)))
		b.WriteString(fmt.Sprintf("[%s](%s)\n\n", EscapeMarkdownV2(d.readLabel(it.Type)), d.itemURL(it)))
	}
	b.WriteString("⚙️ /settings · ⏸ /pause")
	return b.String()
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
	b.WriteString("🗞 *Báo Thể Ích · Bản tin cá nhân*\n\n")
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

// itemURL builds the canonical BaoTheX URL. Every supported content type uses
// the same detail route so Telegram links never land on retired legacy pages.
func (d *Digest) itemURL(it domain.ContentItem) string {
	return fmt.Sprintf("%s/noi-dung/%d", d.baseURL, it.ID)
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n])) + "…"
}
