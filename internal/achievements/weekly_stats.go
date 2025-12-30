package achievements

type WeeklyUserStats struct {
	UserID   string
	Username string

	// 🎧 Voice
	VoiceSeconds int64
	VoiceJoins   int

	// 🎧 Voice channels
	AFKSeconds int64

	// 🎮 Games
	GameSeconds       int64
	GameSessionsCount int
	DistinctGames     int

	// 🎮 Max single game
	MaxSingleGame    string
	MaxSingleGameSec int64
}
