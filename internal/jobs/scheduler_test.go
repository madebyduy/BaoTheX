package jobs

import (
	"testing"

	"repwire/internal/domain"
)

func TestRankHotTopicsPicksTheHottest(t *testing.T) {
	got, ok := rankHotTopics([]domain.HotTopicCluster{
		{
			ClusterID:      1,
			Titles:         []string{"Kết quả vòng đấu: Arsenal thắng 2-0"},
			SourceCount:    4,
			QualitySources: 3,
			Velocity6h:     2,
		},
		{
			ClusterID:      2,
			Titles:         []string{"Trọng tài rút thẻ đỏ gây tranh cãi, HLV chỉ trích VAR"},
			SourceCount:    5,
			QualitySources: 4,
			Velocity6h:     5,
		},
	})
	if !ok {
		t.Fatal("expected a winner")
	}
	if got.ClusterID != 2 {
		t.Fatalf("picked cluster %d, want the disputed one (2)", got.ClusterID)
	}
	if got.Controversy == 0 {
		t.Fatal("winner should carry a controversy score")
	}
	if len(got.Terms) == 0 {
		t.Fatal("winner should record why it was picked")
	}
}

func TestRankHotTopicsDeclinesOnAQuietDay(t *testing.T) {
	// Two mid-tier outlets on a routine result must NOT trigger a piece. A quiet
	// day should cost nothing rather than force an article nobody needs.
	_, ok := rankHotTopics([]domain.HotTopicCluster{
		{
			ClusterID:      1,
			Titles:         []string{"Lịch thi đấu vòng 12"},
			SourceCount:    2,
			QualitySources: 0,
			Velocity6h:     1,
		},
	})
	if ok {
		t.Fatal("a quiet day should not produce a pick")
	}
}

func TestRankHotTopicsHandlesNoContenders(t *testing.T) {
	if _, ok := rankHotTopics(nil); ok {
		t.Fatal("no contenders must not yield a pick")
	}
}

func TestDeskAndPickShareOneScale(t *testing.T) {
	// The bug this guards: the desk once scored with a SQL formula (100-160) while
	// the pick scored with ClusterHeat (0-100), both writing the same column. The
	// day's chosen story showed up as the lowest number on the board.
	contenders := []domain.HotTopicCluster{
		{ClusterID: 1, Titles: []string{"Lịch thi đấu vòng 12"}, SourceCount: 6, QualitySources: 5, Velocity6h: 4},
		{ClusterID: 2, Titles: []string{"Thẻ đỏ gây tranh cãi, HLV chỉ trích trọng tài"}, SourceCount: 4, QualitySources: 4, Velocity6h: 5},
	}
	desk := scoreContenders(contenders)
	winner, ok := rankHotTopics(contenders)
	if !ok {
		t.Fatal("expected a winner")
	}
	var deskScoreOfWinner float64
	for _, p := range desk {
		if p.ClusterID == winner.ClusterID {
			deskScoreOfWinner = p.Heat
		}
	}
	if deskScoreOfWinner != winner.Heat {
		t.Fatalf("desk scored the winner %v but the pick scored it %v — two scales again",
			deskScoreOfWinner, winner.Heat)
	}
	// And the winner must genuinely top the desk's own ranking.
	for _, p := range desk {
		if p.Heat > winner.Heat {
			t.Fatalf("cluster %d scores %v on the desk but %d won with %v",
				p.ClusterID, p.Heat, winner.ClusterID, winner.Heat)
		}
	}
}

func TestScoreContendersKeepsEveryContender(t *testing.T) {
	// The desk shows the whole board, including stories below the pick bar —
	// filtering is the pick's job, not the ranking's.
	contenders := []domain.HotTopicCluster{
		{ClusterID: 1, Titles: []string{"Lịch thi đấu"}, SourceCount: 2},
		{ClusterID: 2, Titles: []string{"Bê bối doping gây tranh cãi"}, SourceCount: 5, QualitySources: 4, Velocity6h: 5},
	}
	if got := scoreContenders(contenders); len(got) != 2 {
		t.Fatalf("scored %d contenders, want all 2", len(got))
	}
}

func TestRankHotTopicsCarriesClusterThrough(t *testing.T) {
	// The claim writes the structural counts back to the database; if the winning
	// cluster were not carried through, those columns would silently store zeros.
	in := domain.HotTopicCluster{
		ClusterID:           7,
		RepresentativeTitle: "Bê bối doping chấn động",
		Titles:              []string{"Bê bối doping chấn động, VĐV bị cấm thi đấu và án phạt gây tranh cãi"},
		SourceCount:         6,
		QualitySources:      4,
		Velocity6h:          6,
		Velocity24h:         11,
		FollowerWeight:      420,
	}
	got, ok := rankHotTopics([]domain.HotTopicCluster{in})
	if !ok {
		t.Fatal("expected a winner")
	}
	if got.Cluster.Velocity24h != 11 || got.Cluster.QualitySources != 4 {
		t.Fatalf("cluster facts lost: %+v", got.Cluster)
	}
	if got.Heat < minDailyPickHeat {
		t.Fatalf("heat %v below the bar %v", got.Heat, minDailyPickHeat)
	}
}
