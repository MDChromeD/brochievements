package storage

import "database/sql"

func StartGameSession(
	db *sql.DB,
	userID, username, game string,
) (int64, error) {

	res, err := db.Exec(`
		INSERT INTO game_sessions (user_id, username, game, started_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`, userID, username, game)

	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func EndGameSession(db *sql.DB, sessionID int64) error {
	_, err := db.Exec(`
		UPDATE game_sessions
		SET ended_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, sessionID)

	return err
}

func CloseUnfinishedGameSessions(store *Storage) error {
	_, err := store.DB.Exec(`
		UPDATE game_sessions
		SET ended_at = CURRENT_TIMESTAMP
		WHERE ended_at IS NULL
	`)
	return err
}
