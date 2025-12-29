package storage

type UserStats struct {
	MessagesCount int
	JoinedAt      string // ← вместо FirstSeen

	VoiceSeconds       int64
	VoiceJoins         int
	FavoriteChannel    string
	FavoriteChannelSec int64

	GameSeconds     int64
	FavoriteGame    string
	FavoriteGameSec int64
	GameSwitches    int
}

func (s *Storage) GetUserStats(userID string) (*UserStats, error) {
	stats := &UserStats{}

	s.fillMessageStats(userID, stats)
	s.fillVoiceStats(userID, stats)
	s.fillFavoriteChannel(userID, stats)
	s.fillGameStats(userID, stats)

	return stats, nil
}
