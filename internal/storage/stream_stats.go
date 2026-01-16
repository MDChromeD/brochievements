package storage

import "database/sql"

type TopStreamTimeResult struct {
	UserID   string
	Username string
	Seconds  int64
}

type TopStreamViewerResult struct {
	UserID   string
	Username string
	Seconds  int64
}

func (s *Storage) GetTopStreamTime() (*TopStreamTimeResult, error) {
	row := s.DB.QueryRow(`
		SELECT
			user_id,
			username,
			SUM(
				strftime('%s', COALESCE(ended_at, CURRENT_TIMESTAMP)) -
				strftime('%s', started_at)
			) AS seconds
		FROM stream_sessions
		GROUP BY user_id
		ORDER BY seconds DESC
		LIMIT 1
	`)

	var res TopStreamTimeResult
	err := row.Scan(&res.UserID, &res.Username, &res.Seconds)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if res.Seconds <= 0 {
		return nil, nil
	}

	return &res, nil
}

func (s *Storage) GetTopStreamViewer() (*TopStreamViewerResult, error) {
	row := s.DB.QueryRow(`
		SELECT
			viewer_id AS user_id,
			viewer_username AS username,
			SUM(
				strftime('%s', COALESCE(v.ended_at, CURRENT_TIMESTAMP)) -
				strftime('%s', v.started_at)
			) AS seconds
		FROM stream_views v
		WHERE v.viewer_id != v.streamer_id
		GROUP BY viewer_id
		ORDER BY seconds DESC
		LIMIT 1
	`)

	var res TopStreamViewerResult
	err := row.Scan(&res.UserID, &res.Username, &res.Seconds)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if res.Seconds <= 0 {
		return nil, nil
	}

	return &res, nil
}
