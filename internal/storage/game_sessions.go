package storage

import (
	"database/sql"
	"time"
)

type GameSession struct {
	ID        int64
	UserID    string
	Game      string
	StartedAt time.Time
}

func (s *Storage) GetActiveGameSession(userID string) (*GameSession, error) {
	row := s.DB.QueryRow(`
		SELECT id, user_id, game, started_at
		FROM game_sessions
		WHERE user_id = ? AND ended_at IS NULL
		LIMIT 1
	`, userID)

	var sess GameSession
	err := row.Scan(&sess.ID, &sess.UserID, &sess.Game, &sess.StartedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Storage) startGameSession(userID, game string) (int64, error) {
	res, err := s.DB.Exec(`
		INSERT INTO game_sessions (user_id, game, started_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`, userID, game)

	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (s *Storage) EndGameSession(sessionID int64) error {
	_, err := s.DB.Exec(`
		UPDATE game_sessions
		SET ended_at = CURRENT_TIMESTAMP
		WHERE id = ? AND ended_at IS NULL
	`, sessionID)

	return err
}

func (s *Storage) EnsureGameSession(userID, game string) error {
	active, err := s.GetActiveGameSession(userID)
	if err != nil {
		return err
	}

	// ▶ нет активной — стартуем
	if active == nil {
		_, err := s.startGameSession(userID, game)
		return err
	}

	// 🔁 та же игра — ничего не делаем
	if active.Game == game {
		return nil
	}

	// 🔄 игра сменилась
	if err := s.EndGameSession(active.ID); err != nil {
		return err
	}

	_, err = s.startGameSession(userID, game)
	return err
}

func (s *Storage) CloseUnfinishedGameSessions() error {
	_, err := s.DB.Exec(`
		UPDATE game_sessions
		SET ended_at = CURRENT_TIMESTAMP
		WHERE ended_at IS NULL
	`)
	return err
}
