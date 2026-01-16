package storage

import "time"

type WeeklyAFKRow struct {
	UserID   string
	Username string
	Seconds  int64
}

func (s *Storage) LoadWeeklyAFKGameRaw(from, to time.Time) ([]WeeklyAFKRow, error) {
	rows, err := s.DB.Query(`
		SELECT
			user_id,
			COALESCE(username, user_id),
			SUM(
				strftime('%s', COALESCE(ended_at, CURRENT_TIMESTAMP)) -
				strftime('%s', started_at)
			) AS seconds
		FROM afk_game_sessions
		WHERE started_at >= ? AND started_at < ?
		GROUP BY user_id
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []WeeklyAFKRow
	for rows.Next() {
		var r WeeklyAFKRow
		if err := rows.Scan(&r.UserID, &r.Username, &r.Seconds); err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

type WeeklyVoiceRow struct {
	UserID   string
	Username string
	Seconds  int64
}

func (s *Storage) LoadWeeklyVoiceRaw(from, to time.Time) ([]WeeklyVoiceRow, error) {
	rows, err := s.DB.Query(`
		SELECT
			vcs.user_id,
			COALESCE(u.username, vcs.user_id),
			SUM(
				strftime('%s', COALESCE(vcs.left_at, CURRENT_TIMESTAMP)) -
				strftime('%s', vcs.joined_at)
			) AS seconds
		FROM voice_channel_sessions vcs
		LEFT JOIN users u ON u.user_id = vcs.user_id
		WHERE vcs.joined_at >= ? AND vcs.joined_at < ?
		GROUP BY vcs.user_id
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []WeeklyVoiceRow
	for rows.Next() {
		var r WeeklyVoiceRow
		if err := rows.Scan(&r.UserID, &r.Username, &r.Seconds); err != nil {
			return nil, err
		}
		if r.Seconds > 0 {
			res = append(res, r)
		}
	}
	return res, nil
}

type WeeklyGameRow struct {
	UserID   string
	Username string
	Game     string
	Seconds  int64
}

func (s *Storage) LoadWeeklyGameRaw(from, to time.Time) ([]WeeklyGameRow, error) {
	rows, err := s.DB.Query(`
		SELECT
			gs.user_id,
			gs.game,
			SUM(
				strftime('%s', COALESCE(gs.ended_at, CURRENT_TIMESTAMP)) -
				strftime('%s', gs.started_at)
			) AS seconds
		FROM game_sessions gs
		LEFT JOIN users u ON u.user_id = gs.user_id
		WHERE gs.started_at >= ? AND gs.started_at < ?
		  AND gs.game != 'Custom Status'
		GROUP BY gs.user_id, gs.game
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []WeeklyGameRow
	for rows.Next() {
		var r WeeklyGameRow
		if err := rows.Scan(&r.UserID, &r.Game, &r.Seconds); err != nil {
			return nil, err
		}
		if r.Seconds > 0 {
			res = append(res, r)
		}
	}
	return res, nil
}

type WeeklyStreamRow struct {
	UserID   string
	Username string
	Seconds  int64
}

func (s *Storage) LoadWeeklyStreamRaw(from, to time.Time) ([]WeeklyStreamRow, error) {
	rows, err := s.DB.Query(`
		SELECT
			ss.user_id,
			COALESCE(ss.username, u.username, ss.user_id),
			SUM(
				strftime('%s', COALESCE(ss.ended_at, CURRENT_TIMESTAMP)) -
				strftime('%s', ss.started_at)
			) AS seconds
		FROM stream_sessions ss
		LEFT JOIN users u ON u.user_id = ss.user_id
		WHERE ss.started_at >= ? AND ss.started_at < ?
		GROUP BY ss.user_id
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []WeeklyStreamRow
	for rows.Next() {
		var r WeeklyStreamRow
		if err := rows.Scan(&r.UserID, &r.Username, &r.Seconds); err != nil {
			return nil, err
		}
		if r.Seconds > 0 {
			res = append(res, r)
		}
	}
	return res, nil
}

type WeeklyStreamViewRow struct {
	UserID   string
	Username string
	Seconds  int64
}

func (s *Storage) LoadWeeklyStreamViewRaw(from, to time.Time) ([]WeeklyStreamViewRow, error) {
	rows, err := s.DB.Query(`
		SELECT
			sv.viewer_id,
			COALESCE(u.username, sv.viewer_id),
			SUM(
				strftime('%s', COALESCE(sv.ended_at, CURRENT_TIMESTAMP)) -
				strftime('%s', sv.started_at)
			) AS seconds
		FROM stream_views sv
		LEFT JOIN users u ON u.user_id = sv.viewer_id
		WHERE sv.started_at >= ? AND sv.started_at < ?
		  AND sv.viewer_id != sv.streamer_id
		GROUP BY sv.viewer_id
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []WeeklyStreamViewRow
	for rows.Next() {
		var r WeeklyStreamViewRow
		if err := rows.Scan(&r.UserID, &r.Username, &r.Seconds); err != nil {
			return nil, err
		}
		if r.Seconds > 0 {
			res = append(res, r)
		}
	}
	return res, nil
}

type WeeklyBreadAFKRow struct {
	UserID   string
	Username string
	Seconds  int64
}

func (s *Storage) LoadWeeklyBreadAFKRaw(from, to time.Time, channelName string) ([]WeeklyBreadAFKRow, error) {
	rows, err := s.DB.Query(`
		SELECT
			vcs.user_id,
			COALESCE(u.username, vcs.user_id),
			SUM(
				strftime('%s', COALESCE(vcs.left_at, CURRENT_TIMESTAMP)) -
				strftime('%s', vcs.joined_at)
			) AS seconds
		FROM voice_channel_sessions vcs
		JOIN voice_channels vc ON vc.channel_id = vcs.channel_id
		LEFT JOIN users u ON u.user_id = vcs.user_id
		WHERE vc.channel_name = ?
		  AND vcs.joined_at >= ? AND vcs.joined_at < ?
		GROUP BY vcs.user_id
	`, channelName, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []WeeklyBreadAFKRow
	for rows.Next() {
		var r WeeklyBreadAFKRow
		if err := rows.Scan(&r.UserID, &r.Username, &r.Seconds); err != nil {
			return nil, err
		}
		if r.Seconds > 0 {
			res = append(res, r)
		}
	}
	return res, nil
}
