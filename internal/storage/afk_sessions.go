package storage

import "database/sql"

type AFKGameSession struct {
	ID        int64
	UserID    string
	Username  string
	Game      string
	StartedAt string
	EndedAt   sql.NullString
}

func (s *Storage) GetActiveAFKGameSession(userID string) (*AFKGameSession, error) {
	row := s.DB.QueryRow(`
		SELECT id, user_id, username, game, started_at, ended_at
		FROM afk_game_sessions
		WHERE user_id = ? AND ended_at IS NULL
		LIMIT 1
	`, userID)

	var sess AFKGameSession
	err := row.Scan(
		&sess.ID,
		&sess.UserID,
		&sess.Username,
		&sess.Game,
		&sess.StartedAt,
		&sess.EndedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Storage) StartAFKGameSession(userID, username, game string) error {
	active, err := s.GetActiveAFKGameSession(userID)
	if err != nil {
		return err
	}
	if active != nil {
		return nil
	}

	_, err = s.DB.Exec(`
		INSERT INTO afk_game_sessions (user_id, username, game, started_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`, userID, username, game)

	return err
}

func (s *Storage) EndAFKGameSession(id int64) error {
	_, err := s.DB.Exec(`
		UPDATE afk_game_sessions
		SET ended_at = CURRENT_TIMESTAMP
		WHERE id = ? AND ended_at IS NULL
	`, id)
	return err
}

func (s *Storage) CloseUnfinishedAFKGameSessions() error {
	_, err := s.DB.Exec(`
		UPDATE afk_game_sessions
		SET ended_at = CURRENT_TIMESTAMP
		WHERE ended_at IS NULL
	`)
	return err
}
