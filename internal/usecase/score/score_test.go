package score_test

import (
	"testing"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── ヘルパー ───────────────────────────────────────────────────────────────

func newUsecase() *score.ScoreUsecase {
	return score.NewScoreUsecase()
}

// hour は JST 基準で time.Time を作る（テスト内は UTC 固定で hour をそのまま使う）
func timeAtHour(hour int) time.Time {
	return time.Date(2026, 4, 30, hour, 0, 0, 0, time.UTC)
}

// ─── TestCalculate_GoodConditions ───────────────────────────────────────────

func TestCalculate_GoodConditions(t *testing.T) {
	t.Parallel()

	uc := newUsecase()
	now := timeAtHour(6) // 早朝6時

	// 大潮・満潮が60分前（仕様: 満潮1時間前）
	highTideTime := now.Add(-60 * time.Minute)
	tide := &entity.TideData{
		TideType:  "大潮",
		HighTides: []entity.TideEvent{{Time: highTideTime, Height: 180}},
	}
	weather := &entity.WeatherData{
		WindSpeed: 2.0,  // 穏やか
		Pressure:  1013, // 基準気圧
	}

	result := uc.Calculate(weather, tide, now)

	require.NotNil(t, result)
	assert.Greater(t, result.Total, 60, "良条件なら合計スコアは60超を期待する")
	// 早朝+大潮+穏やか天気の組み合わせは Good 以上（Excellent も許容）
	assert.True(t,
		result.Rank == entity.ScoreRankGood || result.Rank == entity.ScoreRankExcellent,
		"良条件ならランクは Good か Excellent であるべき（実際: %s）", result.Rank,
	)
	assert.Greater(t, result.TideScore, 0)
	assert.Greater(t, result.WeatherScore, 0)
	assert.Greater(t, result.TimeScore, 0)
	assert.NotEmpty(t, result.Explanation)
}

// ─── TestCalculate_PoorConditions ───────────────────────────────────────────

func TestCalculate_PoorConditions(t *testing.T) {
	t.Parallel()

	uc := newUsecase()
	now := timeAtHour(13) // 日中13時（最低時間帯）

	// 潮汐イベントなし（nil tide ではなく空スライス）
	tide := &entity.TideData{
		TideType:  "小潮",
		HighTides: []entity.TideEvent{},
		LowTides:  []entity.TideEvent{},
	}
	weather := &entity.WeatherData{
		WindSpeed: 12.0, // 強風（>10m/s）
		Pressure:  995,  // 低気圧（差18hPa）
	}

	result := uc.Calculate(weather, tide, now)

	require.NotNil(t, result)
	assert.Less(t, result.Total, 40, "悪条件なら合計スコアは40未満を期待する")
}

// ─── TestCalculate_NilInputs ────────────────────────────────────────────────

func TestCalculate_NilInputs(t *testing.T) {
	t.Parallel()

	uc := newUsecase()
	now := timeAtHour(10)

	result := uc.Calculate(nil, nil, now)

	require.NotNil(t, result)
	// nil 時はデフォルト値（half score）が返る
	assert.Equal(t, 17, result.WeatherScore, "weather nil → 17点（35/2）")
	assert.Equal(t, 20, result.TideScore, "tide nil → 20点（40/2）")
	assert.GreaterOrEqual(t, result.Total, 0)
	assert.LessOrEqual(t, result.Total, 100)
}

// ─── TestGetScoreRank ────────────────────────────────────────────────────────

func TestGetScoreRank(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		score    int
		wantRank entity.ScoreRank
	}{
		{name: "80点以上はExcellent", score: 80, wantRank: entity.ScoreRankExcellent},
		{name: "100点はExcellent", score: 100, wantRank: entity.ScoreRankExcellent},
		{name: "60点以上はGood", score: 60, wantRank: entity.ScoreRankGood},
		{name: "79点はGood", score: 79, wantRank: entity.ScoreRankGood},
		{name: "40点以上はFair", score: 40, wantRank: entity.ScoreRankFair},
		{name: "59点はFair", score: 59, wantRank: entity.ScoreRankFair},
		{name: "20点以上はPoor", score: 20, wantRank: entity.ScoreRankPoor},
		{name: "39点はPoor", score: 39, wantRank: entity.ScoreRankPoor},
		{name: "19点以下はBad", score: 19, wantRank: entity.ScoreRankBad},
		{name: "0点はBad", score: 0, wantRank: entity.ScoreRankBad},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := entity.GetScoreRank(tt.score)
			assert.Equal(t, tt.wantRank, got)
		})
	}
}

// ─── TestCalculateTimeScore ──────────────────────────────────────────────────

func TestCalculateTimeScore(t *testing.T) {
	t.Parallel()

	uc := newUsecase()
	// 潮汐・天気は固定（nil → デフォルト値で固定）して時間帯のみ変化させる

	tests := []struct {
		name          string
		hour          int
		wantTimeScore int
	}{
		{name: "早朝4時（境界）", hour: 4, wantTimeScore: 25},
		{name: "早朝6時", hour: 6, wantTimeScore: 25},
		{name: "早朝7時（境界）", hour: 7, wantTimeScore: 25},
		{name: "朝8時（境界）", hour: 8, wantTimeScore: 18},
		{name: "朝10時（境界）", hour: 10, wantTimeScore: 18},
		{name: "昼11時（その他）", hour: 11, wantTimeScore: 8},
		{name: "昼13時", hour: 13, wantTimeScore: 8},
		{name: "午後14時（境界）", hour: 14, wantTimeScore: 15},
		{name: "午後15時（境界）", hour: 15, wantTimeScore: 15},
		{name: "夕方16時（境界）", hour: 16, wantTimeScore: 22},
		{name: "夕方19時（境界）", hour: 19, wantTimeScore: 22},
		{name: "夜20時（境界）", hour: 20, wantTimeScore: 12},
		{name: "夜22時（境界）", hour: 22, wantTimeScore: 12},
		{name: "深夜23時（その他）", hour: 23, wantTimeScore: 8},
		{name: "深夜0時（その他）", hour: 0, wantTimeScore: 8},
		{name: "深夜3時（その他）", hour: 3, wantTimeScore: 8},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			now := timeAtHour(tt.hour)
			result := uc.Calculate(nil, nil, now)
			assert.Equal(t, tt.wantTimeScore, result.TimeScore, "hour=%d のタイムスコア", tt.hour)
		})
	}
}

// ─── TestCalculateWeatherScore ───────────────────────────────────────────────

func TestCalculateWeatherScore(t *testing.T) {
	t.Parallel()

	uc := newUsecase()
	// 時間帯と潮汐は固定（nil tide, 固定時間）して天気スコアのみ検証する

	tests := []struct {
		name             string
		windSpeed        float64
		pressure         float64
		wantWeatherScore int
	}{
		{name: "無風・基準気圧", windSpeed: 0, pressure: 1013, wantWeatherScore: 35},
		{name: "弱風（≤3m/s）", windSpeed: 3.0, pressure: 1013, wantWeatherScore: 35},
		{name: "やや風（>3m/s）", windSpeed: 4.0, pressure: 1013, wantWeatherScore: 33},
		{name: "中風（>5m/s）", windSpeed: 6.0, pressure: 1013, wantWeatherScore: 29},
		{name: "強風（>7m/s）", windSpeed: 8.0, pressure: 1013, wantWeatherScore: 23},
		{name: "暴風（>10m/s）", windSpeed: 11.0, pressure: 1013, wantWeatherScore: 15},
		{name: "気圧差11hPa（>10）", windSpeed: 0, pressure: 1024, wantWeatherScore: 31},
		{name: "気圧差16hPa（>15）", windSpeed: 0, pressure: 1029, wantWeatherScore: 27},
		{name: "暴風＋低気圧（フロア0）", windSpeed: 11.0, pressure: 990, wantWeatherScore: 7},
		{name: "nilと同様の期待値確認用：直接nil渡し", windSpeed: 0, pressure: 0, wantWeatherScore: -1}, // このケースは別扱い
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantWeatherScore == -1 {
				// nil weather の場合は TestCalculate_NilInputs で検証済み。スキップ
				t.Skip("nil weather は別テストで検証")
			}
			weather := &entity.WeatherData{
				WindSpeed: tt.windSpeed,
				Pressure:  tt.pressure,
			}
			result := uc.Calculate(weather, nil, timeAtHour(10))
			assert.Equal(t, tt.wantWeatherScore, result.WeatherScore,
				"windSpeed=%.1f pressure=%.1f", tt.windSpeed, tt.pressure)
		})
	}
}

// ─── TestCalculateTideScore ──────────────────────────────────────────────────

func TestCalculateTideScore(t *testing.T) {
	t.Parallel()

	uc := newUsecase()
	now := timeAtHour(10)
	fixedWeather := &entity.WeatherData{WindSpeed: 0, Pressure: 1013}

	tests := []struct {
		name          string
		tideType      string
		minutesDiff   float64 // イベントとの時間差（分）
		wantTideScore int
	}{
		{name: "30分以内", tideType: "", minutesDiff: 20, wantTideScore: 40},
		{name: "30分ちょうど", tideType: "", minutesDiff: 30, wantTideScore: 40},
		{name: "31分（>30）", tideType: "", minutesDiff: 31, wantTideScore: 35},
		{name: "60分ちょうど", tideType: "", minutesDiff: 60, wantTideScore: 35},
		{name: "61分（>60）", tideType: "", minutesDiff: 61, wantTideScore: 28},
		{name: "90分ちょうど", tideType: "", minutesDiff: 90, wantTideScore: 28},
		{name: "91分（>90）", tideType: "", minutesDiff: 91, wantTideScore: 20},
		{name: "120分ちょうど", tideType: "", minutesDiff: 120, wantTideScore: 20},
		{name: "121分（>120）", tideType: "", minutesDiff: 121, wantTideScore: 10},
		// 大潮ボーナス（キャップ40を超えない）
		{name: "大潮・30分以内 → キャップ40", tideType: "大潮", minutesDiff: 20, wantTideScore: 40},
		{name: "大潮・60分以内 → 35+5=40", tideType: "大潮", minutesDiff: 45, wantTideScore: 40},
		{name: "大潮・90分以内 → 28+5=33", tideType: "大潮", minutesDiff: 75, wantTideScore: 33},
		{name: "大潮・120分以内 → 20+5=25", tideType: "大潮", minutesDiff: 110, wantTideScore: 25},
		{name: "大潮・>120分 → 10+5=15", tideType: "大潮", minutesDiff: 200, wantTideScore: 15},
		// 中潮ボーナス
		{name: "中潮・90分以内 → 28+3=31", tideType: "中潮", minutesDiff: 75, wantTideScore: 31},
		{name: "中潮・>120分 → 10+3=13", tideType: "中潮", minutesDiff: 200, wantTideScore: 13},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			eventTime := now.Add(time.Duration(-tt.minutesDiff) * time.Minute)
			tide := &entity.TideData{
				TideType:  tt.tideType,
				HighTides: []entity.TideEvent{{Time: eventTime, Height: 150}},
			}
			result := uc.Calculate(fixedWeather, tide, now)
			assert.Equal(t, tt.wantTideScore, result.TideScore,
				"tideType=%s minutesDiff=%.0f", tt.tideType, tt.minutesDiff)
		})
	}
}

// ─── TestCalculate_TotalClamping ─────────────────────────────────────────────

func TestCalculate_TotalClamping(t *testing.T) {
	t.Parallel()

	uc := newUsecase()

	t.Run("合計は100を超えない", func(t *testing.T) {
		// 最良条件：早朝4時 + 大潮 + 30分以内 + 無風基準気圧
		now := timeAtHour(4)
		eventTime := now.Add(-10 * time.Minute)
		tide := &entity.TideData{
			TideType:  "大潮",
			HighTides: []entity.TideEvent{{Time: eventTime, Height: 200}},
		}
		weather := &entity.WeatherData{WindSpeed: 0, Pressure: 1013}

		result := uc.Calculate(weather, tide, now)
		assert.LessOrEqual(t, result.Total, 100, "合計スコアは100以下")
	})

	t.Run("合計は0未満にならない", func(t *testing.T) {
		// 最悪条件：暴風・極低気圧
		now := timeAtHour(13)
		tide := &entity.TideData{
			TideType:  "小潮",
			HighTides: []entity.TideEvent{},
			LowTides:  []entity.TideEvent{},
		}
		weather := &entity.WeatherData{WindSpeed: 20.0, Pressure: 960}

		result := uc.Calculate(weather, tide, now)
		assert.GreaterOrEqual(t, result.Total, 0, "合計スコアは0以上")
	})
}

// ─── TestCalculate_RankMatchesTotal ──────────────────────────────────────────

func TestCalculate_RankMatchesTotal(t *testing.T) {
	t.Parallel()

	uc := newUsecase()
	now := timeAtHour(10)

	result := uc.Calculate(nil, nil, now)

	// Rank が Total から導出されていることを確認
	expectedRank := entity.GetScoreRank(result.Total)
	assert.Equal(t, expectedRank, result.Rank, "Rank は Total から正しく導出されるべき")
}

// ─── TestCalculate_EmptyTideEvents ───────────────────────────────────────────

func TestCalculate_EmptyTideEvents(t *testing.T) {
	t.Parallel()

	uc := newUsecase()
	now := timeAtHour(10)

	// イベントが空スライス（nil でない）の場合
	tide := &entity.TideData{
		TideType:  "小潮",
		HighTides: []entity.TideEvent{},
		LowTides:  []entity.TideEvent{},
	}
	weather := &entity.WeatherData{WindSpeed: 0, Pressure: 1013}

	result := uc.Calculate(weather, tide, now)

	require.NotNil(t, result)
	// イベントなしの場合は最小スコア10点
	assert.Equal(t, 10, result.TideScore, "イベントなしの場合は10点（デフォルト>120分扱い）")
}

// ─── TestCalculate_LowTidesOnly ──────────────────────────────────────────────

func TestCalculate_LowTidesOnly(t *testing.T) {
	t.Parallel()

	uc := newUsecase()
	now := timeAtHour(10)

	// 干潮イベントのみ（高潮イベントなし）
	eventTime := now.Add(-25 * time.Minute)
	tide := &entity.TideData{
		TideType:  "",
		HighTides: []entity.TideEvent{},
		LowTides:  []entity.TideEvent{{Time: eventTime, Height: 20}},
	}
	weather := &entity.WeatherData{WindSpeed: 0, Pressure: 1013}

	result := uc.Calculate(weather, tide, now)

	// 干潮イベントの25分差 → 40点（≤30分）
	assert.Equal(t, 40, result.TideScore, "干潮イベントも正しく参照されるべき")
}

// ─── TestCalculate_ExplanationContent ────────────────────────────────────────

func TestCalculate_ExplanationContent(t *testing.T) {
	t.Parallel()

	uc := newUsecase()
	now := timeAtHour(6)

	eventTime := now.Add(-20 * time.Minute)
	tide := &entity.TideData{
		TideType:  "大潮",
		HighTides: []entity.TideEvent{{Time: eventTime, Height: 180}},
	}
	weather := &entity.WeatherData{
		WindSpeed: 2.5,
		Pressure:  1013,
	}

	result := uc.Calculate(weather, tide, now)

	assert.Contains(t, result.Explanation, "大潮", "Explanation に潮回りを含む")
	assert.Contains(t, result.Explanation, "2.5", "Explanation に風速を含む")
}
