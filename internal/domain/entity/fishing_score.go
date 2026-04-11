package entity

type ScoreRank string

const (
	ScoreRankExcellent ScoreRank = "excellent"
	ScoreRankGood      ScoreRank = "good"
	ScoreRankFair      ScoreRank = "fair"
	ScoreRankPoor      ScoreRank = "poor"
	ScoreRankBad       ScoreRank = "bad"
)

type FishingScore struct {
	Total        int
	Rank         ScoreRank
	TideScore    int
	WeatherScore int
	TimeScore    int
	Explanation  string
}

func GetScoreRank(score int) ScoreRank {
	switch {
	case score >= 80:
		return ScoreRankExcellent
	case score >= 60:
		return ScoreRankGood
	case score >= 40:
		return ScoreRankFair
	case score >= 20:
		return ScoreRankPoor
	default:
		return ScoreRankBad
	}
}
