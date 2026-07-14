package process

import (
	"regexp"
	"strings"

	"repwire/internal/domain"
)

// EntityMatch is an entity detected in an item's text.
type EntityMatch struct {
	EntityID int64
	Role     string
}

// aliasRegexCache memoises compiled word-boundary regexes per alias.
var aliasRegexCache = map[string]*regexp.Regexp{}

// ExtractEntities matches entity aliases against title + summary using
// word-boundary matching (spec section 12), avoiding loose substring hits.
func ExtractEntities(item *domain.ContentItem, extraText string, entities []domain.Entity) []EntityMatch {
	text := strings.Join([]string{item.Title, deref(item.Excerpt), deref(item.Summary), extraText}, " ")

	seen := map[int64]bool{}
	var out []EntityMatch
	for _, e := range entities {
		for _, alias := range e.Aliases {
			alias = strings.TrimSpace(alias)
			if len(alias) < 3 {
				continue // too short to match safely
			}
			re := aliasRegex(alias)
			if re.MatchString(text) {
				if !seen[e.ID] {
					out = append(out, EntityMatch{EntityID: e.ID, Role: "mentioned"})
					seen[e.ID] = true
				}
				break
			}
		}
	}
	return out
}

func aliasRegex(alias string) *regexp.Regexp {
	if re, ok := aliasRegexCache[alias]; ok {
		return re
	}
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(alias) + `\b`)
	aliasRegexCache[alias] = re
	return re
}
