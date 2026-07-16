package process

import (
	"slices"
	"testing"
)

func TestNormalizeForMatchPadsAndStripsPunctuation(t *testing.T) {
	got := normalizeForMatch("Thẻ đỏ gây TRANH CÃI: Onana bị đuổi!")
	want := " thẻ đỏ gây tranh cãi onana bị đuổi "
	if got != want {
		t.Fatalf("normalizeForMatch = %q, want %q", got, want)
	}
}

func TestNormalizeForMatchPreservesVietnameseLetters(t *testing.T) {
	// Diacritics must survive: the whole lexicon depends on it.
	if got := normalizeForMatch("Đường"); got != " đường " {
		t.Fatalf("normalizeForMatch = %q, want %q", got, " đường ")
	}
}

func TestDetectHeatMatchesWholeWordsOnly(t *testing.T) {
	// "var" is a real refereeing term; "variable" and "varane" must not trip it.
	// This is the failure mode that would quietly poison every ranking.
	h := DetectHeat([]string{"Varane and the variable form of Vardy"})
	if slices.Contains(h.Terms, "var") {
		t.Fatalf("matched 'var' inside a longer word: terms=%v", h.Terms)
	}
	h = DetectHeat([]string{"Goal awarded after VAR check"})
	if !slices.Contains(h.Terms, "var") {
		t.Fatalf("did not match standalone 'VAR': terms=%v", h.Terms)
	}
}

func TestDetectHeatMatchesVietnamesePhrases(t *testing.T) {
	h := DetectHeat([]string{"Trọng tài rút thẻ đỏ gây tranh cãi ở phút 90"})
	for _, want := range []string{"thẻ đỏ", "tranh cãi"} {
		if !slices.Contains(h.Terms, want) {
			t.Fatalf("missing %q in terms=%v", want, h.Terms)
		}
	}
	if h.Controversy == 0 {
		t.Fatal("controversy score should be non-zero")
	}
}

func TestDetectHeatDedupesAcrossSyndicatedTitles(t *testing.T) {
	// Ten outlets repeating one headline must not score higher than one outlet
	// saying it once — that is exactly the copy-paste inflation we reject.
	one := DetectHeat([]string{"Pha bóng gây tranh cãi"})
	many := DetectHeat([]string{
		"Pha bóng gây tranh cãi",
		"Pha bóng gây tranh cãi",
		"Pha bóng gây tranh cãi",
	})
	if one.Controversy != many.Controversy {
		t.Fatalf("repeated headline inflated score: one=%v many=%v", one.Controversy, many.Controversy)
	}
}

func TestDetectHeatSeparatesControversyFromAction(t *testing.T) {
	action := DetectHeat([]string{"Man Utd sacked their manager"})
	if action.Action == 0 {
		t.Fatal("expected action signal for 'sacked'")
	}
	if action.Controversy != 0 {
		t.Fatalf("plain transfer news should carry no controversy: %v", action.Terms)
	}
}

func TestDetectHeatCapsRunawayScores(t *testing.T) {
	h := DetectHeat([]string{
		"tranh cãi chỉ trích phẫn nộ bức xúc phản đối bê bối doping gian lận thẻ đỏ án phạt",
	})
	if h.Controversy > controversyCap {
		t.Fatalf("controversy %v exceeded cap %v", h.Controversy, controversyCap)
	}
}

func TestDetectHeatIgnoresEmptyTitles(t *testing.T) {
	h := DetectHeat([]string{"", "   ", "!!!"})
	if h.Total() != 0 || len(h.Terms) != 0 {
		t.Fatalf("expected zero heat, got %+v", h)
	}
}

func TestClusterHeatRewardsQualityOverVolume(t *testing.T) {
	// Two reliable outlets must beat six tabloids on the same story. This is the
	// editorial rule the whole daily pick rests on.
	reliable, _ := ClusterHeat(ClusterHeatInput{
		Titles:         []string{"Báo cáo trọng tài gây tranh cãi"},
		SourceCount:    2,
		QualitySources: 2,
		Velocity6h:     2,
	})
	tabloids, _ := ClusterHeat(ClusterHeatInput{
		Titles:         []string{"Báo cáo trọng tài gây tranh cãi"},
		SourceCount:    6,
		QualitySources: 0,
		Velocity6h:     2,
	})
	if reliable <= tabloids {
		t.Fatalf("quality sources did not outrank volume: reliable=%v tabloids=%v", reliable, tabloids)
	}
}

func TestClusterHeatBreaksTiesOnControversy(t *testing.T) {
	base := ClusterHeatInput{SourceCount: 4, QualitySources: 3, Velocity6h: 3}
	routine := base
	routine.Titles = []string{"Kết quả vòng 12 Ngoại hạng Anh"}
	disputed := base
	disputed.Titles = []string{"Bàn thắng gây tranh cãi sau khi VAR can thiệp"}

	routineScore, _ := ClusterHeat(routine)
	disputedScore, _ := ClusterHeat(disputed)
	if disputedScore <= routineScore {
		t.Fatalf("controversy failed to break the tie: routine=%v disputed=%v", routineScore, disputedScore)
	}
}

func TestClusterHeatIsZeroForNothing(t *testing.T) {
	if score, _ := ClusterHeat(ClusterHeatInput{}); score != 0 {
		t.Fatalf("empty cluster scored %v, want 0", score)
	}
}
