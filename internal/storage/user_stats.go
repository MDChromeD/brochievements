package storage

import "time"

type UserStats struct {
	MessagesCount int
	JoinedAt      string

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
	return s.getUserStatsWithPeriod(userID, "")
}

func (s *Storage) GetUserStatsWeekly(userID string) (*UserStats, error) {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := now.AddDate(0, 0, -(weekday - 1)).Truncate(24 * time.Hour)

	return s.getUserStatsWithPeriod(userID, weekStart.Format("2006-01-02 15:04:05"))
}

func (s *Storage) getUserStatsWithPeriod(userID string, startDate string) (*UserStats, error) {
	stats := &UserStats{}

	s.fillMessageStats(userID, stats, startDate)
	s.fillVoiceStats(userID, stats, startDate)
	s.fillFavoriteChannel(userID, stats, startDate)
	s.fillGameStats(userID, stats, startDate)

	return stats, nil
}

func (s *Storage) fillMessageStats(userID string, stats *UserStats, startDate string) {
	if startDate == "" {
		s.DB.QueryRow(
			`SELECT COUNT(*) FROM messages WHERE user_id = ?`,
			userID,
		).Scan(&stats.MessagesCount)
	} else {
		s.DB.QueryRow(
			`SELECT COUNT(*) FROM messages WHERE user_id = ? AND timestamp >= ?`,
			userID, startDate,
		).Scan(&stats.MessagesCount)
	}

	s.DB.QueryRow(
		`SELECT joined_at FROM users WHERE user_id = ?`,
		userID,
	).Scan(&stats.JoinedAt)
}

func (s *Storage) fillVoiceStats(userID string, stats *UserStats, startDate string) {
	if startDate == "" {
		s.DB.QueryRow(`
			SELECT 
				IFNULL(SUM(strftime('%s', left_at) - strftime('%s', joined_at)), 0),
				COUNT(*)
			FROM voice_sessions
			WHERE user_id = ? AND left_at IS NOT NULL
		`, userID).Scan(&stats.VoiceSeconds, &stats.VoiceJoins)
	} else {
		s.DB.QueryRow(`
			SELECT 
				IFNULL(SUM(strftime('%s', left_at) - strftime('%s', joined_at)), 0),
				COUNT(*)
			FROM voice_sessions
			WHERE user_id = ? AND left_at IS NOT NULL AND joined_at >= ?
		`, userID, startDate).Scan(&stats.VoiceSeconds, &stats.VoiceJoins)
	}
}

func (s *Storage) fillFavoriteChannel(userID string, stats *UserStats, startDate string) {
	if startDate == "" {
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
	} else {
		s.DB.QueryRow(`
			SELECT vc.channel_name,
			       SUM(strftime('%s', vcs.left_at) - strftime('%s', vcs.joined_at)) AS seconds
			FROM voice_channel_sessions vcs
			JOIN voice_channels vc ON vc.channel_id = vcs.channel_id
			WHERE vcs.user_id = ? AND vcs.left_at IS NOT NULL AND vcs.joined_at >= ?
			GROUP BY vcs.channel_id
			ORDER BY seconds DESC
			LIMIT 1
		`, userID, startDate).Scan(&stats.FavoriteChannel, &stats.FavoriteChannelSec)
	}
}

func (s *Storage) fillGameStats(userID string, stats *UserStats, startDate string) {
	if startDate == "" {
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
	} else {
		s.DB.QueryRow(`
			SELECT IFNULL(SUM(strftime('%s', ended_at) - strftime('%s', started_at)), 0)
			FROM game_sessions
			WHERE user_id = ? AND ended_at IS NOT NULL AND started_at >= ?
		`, userID, startDate).Scan(&stats.GameSeconds)

		s.DB.QueryRow(`
			SELECT game,
			       SUM(strftime('%s', ended_at) - strftime('%s', started_at)) AS seconds
			FROM game_sessions
			WHERE user_id = ? AND ended_at IS NOT NULL AND started_at >= ?
			GROUP BY game
			ORDER BY seconds DESC
			LIMIT 1
		`, userID, startDate).Scan(&stats.FavoriteGame, &stats.FavoriteGameSec)

		s.DB.QueryRow(`
			SELECT COUNT(*) FROM game_sessions WHERE user_id = ? AND started_at >= ?
		`, userID, startDate).Scan(&stats.GameSwitches)
	}
}
