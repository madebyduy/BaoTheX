package process

import (
	"regexp"
	"strings"
	"sync"

	"repwire/internal/domain"
)

// EntityMatch is an entity detected in an item's text.
type EntityMatch struct {
	EntityID int64
	Role     string
}

// aliasRegexCache memoises compiled word-boundary regexes per alias. It is read
// and written from every job handler, and the worker runs them concurrently, so
// it must be a sync.Map — a plain map here is a data race that surfaces as a
// "concurrent map writes" panic under load rather than anything diagnosable.
var aliasRegexCache sync.Map // string -> *regexp.Regexp

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
			if aliasRegex(alias).MatchString(text) {
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

// boundary is a Unicode-aware stand-in for \b.
//
// Go's \b is defined on ASCII word characters only, so it cannot close a match
// on a Vietnamese alias: "Quỷ đỏ" ends in "ỏ", which \b does not consider a word
// character, and the pattern silently never matches. On a Vietnamese sports
// desk that is most of the interesting vocabulary — "Quỷ đỏ", "Pháo thủ",
// "Lữ đoàn đỏ", "Bà đầm già" all failed while their English names worked, so
// entity extraction quietly under-reported exactly the local coverage it was
// meant to connect.
//
// Go's regexp has no lookaround, so the boundary is expressed as an alternation
// that consumes one neighbouring rune. That is harmless here: callers only ask
// whether the alias appears at all, never where.
const boundary = `[^\p{L}\p{N}_]`

func aliasRegex(alias string) *regexp.Regexp {
	if cached, ok := aliasRegexCache.Load(alias); ok {
		return cached.(*regexp.Regexp)
	}
	re := regexp.MustCompile(`(?i)(^|` + boundary + `)` + regexp.QuoteMeta(alias) + `($|` + boundary + `)`)
	actual, _ := aliasRegexCache.LoadOrStore(alias, re)
	return actual.(*regexp.Regexp)
}
