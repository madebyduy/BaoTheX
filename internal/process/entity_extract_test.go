package process

import (
	"slices"
	"testing"

	"repwire/internal/domain"
)

// The aliases below mirror migration 0023. They are duplicated here on purpose:
// these tests exist to prove the alias *rules* hold on real headlines, and the
// rules are what the migration's comment asks future editors to follow. A false
// match is worse than a missed one — a wrong entity silently corrupts clustering
// for every article it touches.
func sportsEntities() []domain.Entity {
	return []domain.Entity{
		{ID: 1, Slug: "manchester-united", Aliases: []string{"Manchester United", "Man United", "Man Utd", "Quỷ đỏ"}},
		{ID: 2, Slug: "manchester-city", Aliases: []string{"Manchester City", "Man City"}},
		{ID: 3, Slug: "real-madrid", Aliases: []string{"Real Madrid", "Kền kền trắng", "Los Blancos"}},
		{ID: 4, Slug: "inter-milan", Aliases: []string{"Inter Milan", "Nerazzurri"}},
		{ID: 5, Slug: "as-roma", Aliases: []string{"AS Roma", "Roma"}},
		{ID: 6, Slug: "lionel-messi", Aliases: []string{"Lionel Messi", "Messi"}},
		{ID: 7, Slug: "premier-league", Aliases: []string{"Premier League", "Ngoại hạng Anh", "EPL"}},
		{ID: 8, Slug: "vietnam-nt", Aliases: []string{"đội tuyển Việt Nam", "tuyển Việt Nam", "ĐT Việt Nam"}},
		{ID: 9, Slug: "quang-hai", Aliases: []string{"Nguyễn Quang Hải", "Quang Hải"}},
	}
}

func extractSlugs(t *testing.T, title string) []string {
	t.Helper()
	item := &domain.ContentItem{Title: title}
	matches := ExtractEntities(item, "", sportsEntities())
	all := sportsEntities()
	var slugs []string
	for _, m := range matches {
		for _, e := range all {
			if e.ID == m.EntityID {
				slugs = append(slugs, e.Slug)
			}
		}
	}
	slices.Sort(slugs)
	return slugs
}

func TestExtractEntitiesMatchesEnglishAndVietnameseAlike(t *testing.T) {
	// The whole point of entity clustering: the same event in two languages must
	// resolve to the same entities, because trigram similarity between these two
	// headlines is essentially zero.
	english := extractSlugs(t, "Manchester United and Messi dominate the Premier League")
	vietnamese := extractSlugs(t, "Quỷ đỏ cùng Messi thống trị Ngoại hạng Anh")
	if !slices.Equal(english, vietnamese) {
		t.Fatalf("cross-language mismatch:\n  en=%v\n  vi=%v", english, vietnamese)
	}
	if len(english) != 3 {
		t.Fatalf("expected 3 entities, got %v", english)
	}
}

func TestExtractEntitiesRejectsAmbiguousSubstrings(t *testing.T) {
	// Each of these is a headline that a careless alias would ruin. "City",
	// "United", "Real" and "Inter" are never aliases on their own for exactly
	// this reason.
	for _, title := range []string{
		"New York City hosts the final",
		"The United States announces its squad",
		"A real turning point for the season",
		"Internet outage delays kickoff",
		"Romania qualify for the play-offs",
		"Milanese fans travel in numbers",
	} {
		if got := extractSlugs(t, title); len(got) != 0 {
			t.Fatalf("false positive on %q: matched %v", title, got)
		}
	}
}

func TestExtractEntitiesStillMatchesTheQualifiedForms(t *testing.T) {
	// The flip side of the rule above: qualifying the alias must not break the
	// real mention.
	if got := extractSlugs(t, "Man City beat Man Utd at the Etihad"); len(got) != 2 {
		t.Fatalf("expected both clubs, got %v", got)
	}
	if got := extractSlugs(t, "Real Madrid sign a new keeper"); !slices.Contains(got, "real-madrid") {
		t.Fatalf("expected real-madrid, got %v", got)
	}
	if got := extractSlugs(t, "AS Roma host Inter Milan"); len(got) != 2 {
		t.Fatalf("expected Roma and Inter, got %v", got)
	}
}

func TestExtractEntitiesIsCaseInsensitive(t *testing.T) {
	if got := extractSlugs(t, "MAN UTD SACK THEIR MANAGER"); !slices.Contains(got, "manchester-united") {
		t.Fatalf("uppercase headline missed: %v", got)
	}
}

func TestExtractEntitiesHandlesVietnameseDiacritics(t *testing.T) {
	got := extractSlugs(t, "Quang Hải trở lại đội tuyển Việt Nam")
	for _, want := range []string{"quang-hai", "vietnam-nt"} {
		if !slices.Contains(got, want) {
			t.Fatalf("missing %q in %v", want, got)
		}
	}
}

func TestExtractEntitiesDeduplicatesPerEntity(t *testing.T) {
	// Two aliases of one club in one headline is still one entity, or shared
	// entity counts inflate and unrelated stories merge.
	got := extractSlugs(t, "Manchester United confirm it: Man Utd sign a striker")
	if len(got) != 1 || got[0] != "manchester-united" {
		t.Fatalf("expected exactly one entity, got %v", got)
	}
}

func TestExtractEntitiesFindsNothingInNonSportsText(t *testing.T) {
	if got := extractSlugs(t, "Protein synthesis rates after resistance training"); len(got) != 0 {
		t.Fatalf("fitness text matched sports entities: %v", got)
	}
}
