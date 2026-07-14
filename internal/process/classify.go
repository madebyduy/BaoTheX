package process

import (
	"math"
	"sort"
	"strings"

	"repwire/internal/domain"
)

// Assignment is a topic classification result for a content item.
type Assignment struct {
	TopicID    int64
	Confidence float64
	IsPrimary  bool
}

// Classify assigns topics to an item using rule-based keyword matching
// (spec section 12). Title matches are weighted more heavily than body matches.
func Classify(item *domain.ContentItem, extraText string, topics []domain.Topic) []Assignment {
	title := strings.ToLower(item.Title)
	body := strings.ToLower(strings.Join([]string{item.Title, deref(item.Excerpt), deref(item.Summary), extraText}, " "))

	var out []Assignment
	for _, t := range topics {
		var hits, weight float64
		for _, kw := range t.Keywords {
			k := strings.ToLower(strings.TrimSpace(kw))
			if k == "" {
				continue
			}
			if strings.Contains(title, k) {
				weight += 3 // title match weighs heavily
				hits++
			} else if strings.Contains(body, k) {
				weight += 1
				hits++
			}
		}
		if hits == 0 {
			continue
		}
		conf := math.Min(weight/6.0, 1.0)
		if conf >= 0.3 {
			out = append(out, Assignment{TopicID: t.ID, Confidence: conf})
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Confidence > out[j].Confidence })
	if len(out) > 5 {
		out = out[:5]
	}
	if len(out) > 0 {
		out[0].IsPrimary = true
	}
	return out
}

// ToContentTopics converts assignments to persistence rows for a content id.
func ToContentTopics(contentID int64, assignments []Assignment) []domain.ContentTopic {
	rows := make([]domain.ContentTopic, len(assignments))
	for i, a := range assignments {
		rows[i] = domain.ContentTopic{
			ContentID:  contentID,
			TopicID:    a.TopicID,
			Confidence: a.Confidence,
			IsPrimary:  a.IsPrimary,
		}
	}
	return rows
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
