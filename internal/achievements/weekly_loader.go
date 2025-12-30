package achievements

import (
	"brochievements/internal/storage"
	"time"
)

func lastWeekRange() (from, to time.Time) {
	now := time.Now()
	to = now
	from = now.AddDate(0, 0, -7)
	return
}

func LoadWeeklyStats(store *storage.Storage) ([]WeeklyUserStats, error) {
	from, to := lastWeekRange()

	rows, err := store.DB.Query(`
		SELECT
			u.user_id,
			u.username,

			-- voice
			IFNULL(SUM(strftime('%s', vs.left_at) - strftime('%s', vs.joined_at)), 0) AS voice_seconds,
			COUNT(vs.id) AS voice_joins,

			-- games
			IFNULL(SUM(strftime('%s', gs.ended_at) - strftime('%s', gs.started_at)), 0) AS game_seconds,
			COUNT(gs.id) AS game_sessions,
			COUNT(DISTINCT gs.game) AS distinct_games

		FROM users u
		LEFT JOIN voice_sessions vs
			ON vs.user_id = u.user_id
			AND vs.joined_at >= ?
			AND vs.left_at IS NOT NULL

		LEFT JOIN game_sessions gs
			ON gs.user_id = u.user_id
			AND gs.started_at >= ?
			AND gs.ended_at IS NOT NULL

		GROUP BY u.user_id
	`, to, from)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []WeeklyUserStats

	for rows.Next() {
		var s WeeklyUserStats
		err := rows.Scan(
			&s.UserID,
			&s.Username,
			&s.VoiceSeconds,
			&s.VoiceJoins,
			&s.GameSeconds,
			&s.GameSessionsCount,
			&s.DistinctGames,
		)
		if err != nil {
			return nil, err
		}

		result = append(result, s)
	}

	return result, nil
}

func LoadWeeklyAFK(store *storage.Storage, stats []WeeklyUserStats) error {
	from, _ := lastWeekRange()

	rows, err := store.DB.Query(`
		SELECT
			vcs.user_id,
			IFNULL(SUM(strftime('%s', vcs.left_at) - strftime('%s', vcs.joined_at)), 0)
		FROM voice_channel_sessions vcs
		JOIN voice_channels vc ON vc.channel_id = vcs.channel_id
		WHERE vc.channel_name = 'Я_за_хлебушком_пацаны'
		  AND vcs.joined_at >= ?
		  AND vcs.left_at IS NOT NULL
		GROUP BY vcs.user_id
	`, from)

	if err != nil {
		return err
	}
	defer rows.Close()

	afkMap := map[string]int64{}

	for rows.Next() {
		var userID string
		var seconds int64
		rows.Scan(&userID, &seconds)
		afkMap[userID] = seconds
	}

	for i := range stats {
		stats[i].AFKSeconds = afkMap[stats[i].UserID]
	}

	return nil
}

func LoadWeeklyMaxGame(store *storage.Storage, stats []WeeklyUserStats) error {
	from, _ := lastWeekRange()

	rows, err := store.DB.Query(`
		SELECT
			user_id,
			game,
			SUM(strftime('%s', ended_at) - strftime('%s', started_at)) AS seconds
		FROM game_sessions
		WHERE started_at >= ?
		  AND ended_at IS NOT NULL
		GROUP BY user_id, game
	`, from)

	if err != nil {
		return err
	}
	defer rows.Close()

	maxMap := map[string]struct {
		game string
		sec  int64
	}{}

	for rows.Next() {
		var userID, game string
		var sec int64
		rows.Scan(&userID, &game, &sec)

		if cur, ok := maxMap[userID]; !ok || sec > cur.sec {
			maxMap[userID] = struct {
				game string
				sec  int64
			}{game, sec}
		}
	}

	for i := range stats {
		if m, ok := maxMap[stats[i].UserID]; ok {
			stats[i].MaxSingleGame = m.game
			stats[i].MaxSingleGameSec = m.sec
		}
	}

	return nil
}
