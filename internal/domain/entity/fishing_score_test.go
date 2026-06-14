package entity

import "testing"

func TestGetScoreRank(t *testing.T) {
	tests := []struct {
		name  string
		score int
		want  ScoreRank
	}{
		{name: "上限100はexcellent", score: 100, want: ScoreRankExcellent},
		{name: "境界80はexcellent", score: 80, want: ScoreRankExcellent},
		{name: "79はgood", score: 79, want: ScoreRankGood},
		{name: "境界60はgood", score: 60, want: ScoreRankGood},
		{name: "59はfair", score: 59, want: ScoreRankFair},
		{name: "境界40はfair", score: 40, want: ScoreRankFair},
		{name: "39はpoor", score: 39, want: ScoreRankPoor},
		{name: "境界20はpoor", score: 20, want: ScoreRankPoor},
		{name: "19はbad", score: 19, want: ScoreRankBad},
		{name: "0はbad", score: 0, want: ScoreRankBad},
		{name: "負値はbad", score: -10, want: ScoreRankBad},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetScoreRank(tt.score); got != tt.want {
				t.Errorf("GetScoreRank(%d) = %q, want %q", tt.score, got, tt.want)
			}
		})
	}
}
