package achievements

import "fmt"

type WeeklyUserStats struct {
	UserID   string
	Username string

	Title string

	Value string
	Extra string

	// 🎧 Voice
	VoiceSeconds int64
	VoiceJoins   int

	// 🎧 Voice channels
	AFKSeconds int64

	// 🎮 Games
	GameSeconds       int64
	AfkGameSeconds    int64
	GameSessionsCount int
	DistinctGames     int

	// 🎮 Max single game
	MaxSingleGame    string
	MaxSingleGameSec int64

	// Streams
	StreamSeconds     int64
	StreamViewSeconds int64
}

type WeeklyStat struct {
	UserID   string
	Username string
	Value    string
}

func formatDuration(seconds int64) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60

	if h > 0 {
		return fmt.Sprintf("%dч %dм", h, m)
	}
	return fmt.Sprintf("%dм", m)
}
