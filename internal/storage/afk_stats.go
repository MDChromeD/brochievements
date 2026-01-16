package storage

import "database/sql"

type AFKFarmerResult struct {
	UserID     string
	Username   string
	AFKSeconds int64
}

func (s *Storage) GetTopAFKFarmer() (*AFKFarmerResult, error) {
	row := s.DB.QueryRow(`
		SELECT
			user_id,
			COALESCE(username, user_id),
			SUM(
				strftime('%s', COALESCE(ended_at, CURRENT_TIMESTAMP)) -
				strftime('%s', started_at)
			) AS afk_seconds
		FROM afk_game_sessions
		GROUP BY user_id
		ORDER BY afk_seconds DESC
		LIMIT 1
	`)

	var res AFKFarmerResult
	err := row.Scan(
		&res.UserID,
		&res.Username,
		&res.AFKSeconds,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &res, nil
}
