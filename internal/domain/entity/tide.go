package entity

import "time"

// TideData は tide736.net の1港・複数日分の潮汐。Next の FormattedTideData と同じ中身。
type TideData struct {
	Port     TidePort
	Days     []DailyTide // 日付の昇順
	DataTime time.Time
}

type TidePort struct {
	PrefectureCode string
	PortCode       string
	PortName       string
}

type DailyTide struct {
	Date  string // YYYY-MM-DD
	Sun   SunInfo
	Moon  MoonInfo
	Tide  []TideEvent // 潮汐曲線
	Flood []TideEvent // 満潮
	Ebb   []TideEvent // 干潮（tide736 の edd）
}

// SunInfo の時刻はすべて tide736 の文字列（HH:mm）のまま。
type SunInfo struct {
	AstroTwilight   []string
	RegularTwilight []string
	Rise            string
	Midline         string
	Set             string
}

// MoonInfo の値は tide736 の文字列のまま。Name は無い日は nil。
type MoonInfo struct {
	Brightness string
	Age        string
	Title      string // 潮回り（大潮・中潮など）
	Rise       string
	Midline    string
	Set        string
	Name       *string
}

type TideEvent struct {
	Time   string  // HH:mm
	UnixMs int64   // Unix 時刻（ミリ秒）
	CM     float64 // 潮位（cm）
}
