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

func (s *Storage) fillMessageStats(userID string, stats *UserStats) {
	s.DB.QueryRow(
		`SELECT COUNT(*) FROM messages WHERE user_id = ?`,
		userID,
	).Scan(&stats.MessagesCount)

	s.DB.QueryRow(
		`SELECT joined_at FROM users WHERE user_id = ?`,
		userID,
	).Scan(&stats.JoinedAt)
}

func (s *Storage) fillVoiceStats(userID string, stats *UserStats) {
	s.DB.QueryRow(`
		SELECT 
			IFNULL(SUM(strftime('%s', left_at) - strftime('%s', joined_at)), 0),
			COUNT(*)
		FROM voice_sessions
		WHERE user_id = ? AND left_at IS NOT NULL
	`, userID).Scan(&stats.VoiceSeconds, &stats.VoiceJoins)
}

func (s *Storage) fillFavoriteChannel(userID string, stats *UserStats) {
	s.DB.QueryRow(`
		SELECT vc.channel_name,
		       SUM(strftime('%s', vcs.left_at) - strftime('%s', vcs.joined_at)) AS seconds
		FROM voice_channel_sessions vcs
		JOIN voice_channels vc ON vc.channel_id = vcs.channel_id
		WHERE vcs.user_id = ? AND vcs.left_at IS NOT NULL
		GROUP BY vcs.channel_id
		ORDER BY seconds DESC
		LIMIT 1
	`, userID).Scan(&stats.FavoriteChannel, &stats.FavoriteChannelSec)
}

func (s *Storage) fillGameStats(userID string, stats *UserStats) {
	s.DB.QueryRow(`
		SELECT IFNULL(SUM(strftime('%s', ended_at) - strftime('%s', started_at)), 0)
		FROM game_sessions
		WHERE user_id = ? AND ended_at IS NOT NULL
	`, userID).Scan(&stats.GameSeconds)

	s.DB.QueryRow(`
		SELECT game,
		       SUM(strftime('%s', ended_at) - strftime('%s', started_at)) AS seconds
		FROM game_sessions
		WHERE user_id = ? AND ended_at IS NOT NULL
		GROUP BY game
		ORDER BY seconds DESC
		LIMIT 1
	`, userID).Scan(&stats.FavoriteGame, &stats.FavoriteGameSec)

	s.DB.QueryRow(`
		SELECT COUNT(*) FROM game_sessions WHERE user_id = ?
	`, userID).Scan(&stats.GameSwitches)
}
