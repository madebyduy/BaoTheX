package process

import (
	"sort"
	"strings"
	"unicode"
)

// This file scores how much a story deserves the newsroom's single daily
// editorial slot. Every signal here is pure text or arithmetic: no LLM call, no
// network. That is the point — ranking must stay free so the LLM budget can be
// spent on the one story that wins, rather than spread thinly over everything
// that happens to arrive.

// Controversy terms mark a story people are arguing about: a decision under
// dispute, an accusation, a punishment. These carry the most editorial weight
// because they are what an analysis piece can actually add value to.
//
// Terms are matched on whole words after normalisation, so "var" matches the
// refereeing system but not "variable". Multi-word phrases are matched as a
// contiguous run of words.
var controversyTerms = []string{
	// Vietnamese
	"tranh cãi", "gây bão", "chỉ trích", "phẫn nộ", "bức xúc", "phản đối",
	"khẩu chiến", "đấu khẩu", "đá xoáy", "mỉa mai", "lùm xùm", "bê bối",
	"nghi vấn", "cáo buộc", "tố cáo", "kiện", "đòi công bằng",
	"doping", "dàn xếp tỷ số", "bán độ", "hối lộ", "gian lận", "thiên vị",
	"việt vị", "thẻ đỏ", "phạt đền", "án phạt", "treo giò", "cấm thi đấu",
	"kỷ luật", "truất quyền", "phân biệt chủng tộc", "miệt thị", "xúc phạm",
	"ẩu đả", "xô xát", "bạo lực", "sai lầm", "oan",
	// English
	"controversy", "controversial", "criticism", "criticised", "criticized",
	"backlash", "outrage", "outrageous", "slammed", "blasted", "furious",
	"row", "feud", "clash", "scandal", "doping", "match fixing", "bribery",
	"offside", "red card", "var", "penalty", "ban", "banned", "suspended",
	"racism", "racist", "abuse", "appeal", "protest", "investigation",
	"charged", "accused", "allegations",
}

// Action terms mark a decisive event: someone left, someone was hired, a record
// fell. They matter less than controversy for an opinion piece — a transfer is
// news, a disputed transfer is a story — so they score lower.
var actionTerms = []string{
	// Vietnamese
	"sa thải", "từ chức", "chia tay", "bổ nhiệm", "thay tướng",
	"bom tấn", "chuyển nhượng", "ký hợp đồng", "kỷ lục", "lịch sử",
	"đăng quang", "lên ngôi", "lội ngược dòng", "gây sốc", "sốc",
	"chấn thương", "giải nghệ", "tái xuất", "trở lại", "thừa nhận",
	"tiết lộ", "tuyên bố", "xác nhận",
	// English
	"sacked", "fired", "resigns", "resigned", "quits", "appointed",
	"transfer", "signs", "signing", "record", "historic", "comeback",
	"shock", "injury", "injured", "retires", "confirms", "reveals",
	"admits", "announces",
}

// HeatSignals is the free, explainable part of a story's heat. Terms is kept so
// the admin desk can show *why* a topic was picked instead of asking an editor
// to trust an opaque number.
type HeatSignals struct {
	Controversy float64
	Action      float64
	Terms       []string
}

// Total is the combined lexical contribution.
func (h HeatSignals) Total() float64 { return h.Controversy + h.Action }

const (
	// A single controversy word is a weak hint; several distinct ones across a
	// cluster's headlines is a strong one. Points are per distinct term and
	// capped so one ranting headline cannot outweigh genuine multi-source noise.
	controversyPerTerm = 6.0
	controversyCap     = 24.0
	actionPerTerm      = 3.0
	actionCap          = 12.0
)

// DetectHeat scores a cluster's headlines for controversy and decisive action.
// Terms are counted once per cluster no matter how many outlets repeat them, so
// syndicated copies of one headline do not inflate the score.
func DetectHeat(titles []string) HeatSignals {
	seenControversy := map[string]bool{}
	seenAction := map[string]bool{}
	for _, title := range titles {
		norm := normalizeForMatch(title)
		if norm == " " {
			continue
		}
		for _, term := range controversyTerms {
			if containsPhrase(norm, term) {
				seenControversy[term] = true
			}
		}
		for _, term := range actionTerms {
			if containsPhrase(norm, term) {
				seenAction[term] = true
			}
		}
	}
	h := HeatSignals{
		Controversy: min(float64(len(seenControversy))*controversyPerTerm, controversyCap),
		Action:      min(float64(len(seenAction))*actionPerTerm, actionCap),
		Terms:       make([]string, 0, len(seenControversy)+len(seenAction)),
	}
	for term := range seenControversy {
		h.Terms = append(h.Terms, term)
	}
	for term := range seenAction {
		h.Terms = append(h.Terms, term)
	}
	sort.Strings(h.Terms)
	return h
}

// ClusterHeatInput carries the structural facts the database already knows
// about a cluster. Nothing here costs an LLM call.
type ClusterHeatInput struct {
	Titles []string
	// SourceCount is the number of distinct outlets covering the story.
	SourceCount int
	// QualitySources counts distinct outlets with quality >= 4. Ten tabloids
	// copying each other must not outrank two serious newspapers.
	QualitySources int
	// Velocity6h is how many pieces landed in the last six hours — the sharpest
	// available proxy for "this is blowing up right now".
	Velocity6h int
	// FollowerWeight is the summed follower count of the cluster's topics,
	// i.e. how many of our readers actually care.
	FollowerWeight int
}

// ClusterHeat combines the free structural and lexical signals into the single
// number the daily pick sorts on.
//
// The weighting reflects editorial judgement rather than arithmetic
// convenience: independent corroboration (quality sources) and live velocity
// dominate, controversy is the tiebreak that decides which of two equally
// well-covered stories is worth an opinion piece, and audience interest only
// nudges. Caps stop any single dimension from running away.
func ClusterHeat(in ClusterHeatInput) (float64, HeatSignals) {
	signals := DetectHeat(in.Titles)
	score := 0.0
	score += min(float64(in.QualitySources)*10, 40)
	score += min(float64(in.SourceCount)*4, 24)
	score += min(float64(in.Velocity6h)*5, 30)
	score += signals.Total()
	score += min(float64(in.FollowerWeight)*0.02, 15)
	return score, signals
}

// containsPhrase reports whether the already-normalised haystack contains term
// as a whole word or a contiguous run of whole words.
func containsPhrase(normalized, term string) bool {
	return strings.Contains(normalized, " "+term+" ")
}

// normalizeForMatch lowercases text and reduces every non-alphanumeric rune to a
// single space, then pads both ends. Padding lets containsPhrase test word
// boundaries with a plain substring search. Vietnamese diacritics are letters to
// unicode.IsLetter and survive untouched, so "thẻ đỏ" stays matchable.
func normalizeForMatch(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	return " " + strings.Join(strings.Fields(b.String()), " ") + " "
}
