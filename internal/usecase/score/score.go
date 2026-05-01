package score

import (
	"fmt"
	"math"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

const (
	maxTideScore    = 40
	maxWeatherScore = 35
	maxTimeScore    = 25
)

// ScoreUsecase は釣りやすさスコアを算出するユースケースなのだ。
type ScoreUsecase struct{}

// NewScoreUsecase は ScoreUsecase を生成するのだ。
func NewScoreUsecase() *ScoreUsecase {
	return &ScoreUsecase{}
}

// Calculate は天気・潮汐・時間帯から FishingScore を算出するのだ。
func (u *ScoreUsecase) Calculate(weather *entity.WeatherData, tide *entity.TideData, now time.Time) *entity.FishingScore {
	tideScore := u.calculateTideScore(tide, now)
	weatherScore := u.calculateWeatherScore(weather)
	timeScore := u.calculateTimeScore(now)

	total := tideScore + weatherScore + timeScore
	if total > 100 {
		total = 100
	}
	if total < 0 {
		total = 0
	}

	return &entity.FishingScore{
		Total:        total,
		Rank:         entity.GetScoreRank(total),
		TideScore:    tideScore,
		WeatherScore: weatherScore,
		TimeScore:    timeScore,
		Explanation:  u.generateExplanation(tideScore, weatherScore, timeScore, tide, weather),
	}
}

// calculateTideScore は潮汐スコアを算出するのだ（最大40点）。
func (u *ScoreUsecase) calculateTideScore(tide *entity.TideData, now time.Time) int {
	if tide == nil {
		return maxTideScore / 2
	}

	allEvents := append(tide.HighTides, tide.LowTides...)

	minMinutes := math.MaxFloat64
	for _, event := range allEvents {
		diff := math.Abs(now.Sub(event.Time).Minutes())
		if diff < minMinutes {
			minMinutes = diff
		}
	}

	var baseScore int
	switch {
	case minMinutes <= 30:
		baseScore = 40
	case minMinutes <= 60:
		baseScore = 35
	case minMinutes <= 90:
		baseScore = 28
	case minMinutes <= 120:
		baseScore = 20
	default:
		baseScore = 10
	}

	bonus := 0
	switch tide.TideType {
	case "大潮":
		bonus = 5
	case "中潮":
		bonus = 3
	}

	result := baseScore + bonus
	if result > maxTideScore {
		result = maxTideScore
	}
	return result
}

// calculateWeatherScore は天気スコアを算出するのだ（最大35点）。
func (u *ScoreUsecase) calculateWeatherScore(weather *entity.WeatherData) int {
	if weather == nil {
		return maxWeatherScore / 2
	}

	s := maxWeatherScore

	switch {
	case weather.WindSpeed > 10:
		s -= 20
	case weather.WindSpeed > 7:
		s -= 12
	case weather.WindSpeed > 5:
		s -= 6
	case weather.WindSpeed > 3:
		s -= 2
	}

	pressureDiff := math.Abs(weather.Pressure - 1013.0)
	switch {
	case pressureDiff > 15:
		s -= 8
	case pressureDiff > 10:
		s -= 4
	}

	if s < 0 {
		s = 0
	}
	return s
}

// calculateTimeScore は時間帯スコアを算出するのだ（最大25点）。
func (u *ScoreUsecase) calculateTimeScore(now time.Time) int {
	hour := now.Hour()
	switch {
	case hour >= 4 && hour <= 7:
		return 25
	case hour >= 16 && hour <= 19:
		return 22
	case hour >= 8 && hour <= 10:
		return 18
	case hour >= 14 && hour <= 15:
		return 15
	case hour >= 20 && hour <= 22:
		return 12
	default:
		return 8
	}
}

// generateExplanation はスコアの簡易説明文を生成するのだ。
func (u *ScoreUsecase) generateExplanation(
	tideScore, weatherScore, timeScore int,
	tide *entity.TideData,
	weather *entity.WeatherData,
) string {
	explanation := fmt.Sprintf("釣りやすさスコア：潮汐%d点＋天気%d点＋時間帯%d点", tideScore, weatherScore, timeScore)

	if tide != nil && tide.TideType != "" {
		explanation += fmt.Sprintf("。潮回り：%s", tide.TideType)
	}
	if weather != nil {
		explanation += fmt.Sprintf("。風速：%.1fm/s", weather.WindSpeed)
	}

	return explanation
}
