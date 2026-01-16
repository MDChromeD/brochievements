package storage

import "time"

func (s *Storage) UpsertUser(
	userID string,
	username string,
	joinedAt *time.Time,
) error {

	if userID == "" {
		return nil
	}

	_, err := s.DB.Exec(`
		INSERT INTO users (user_id, username, joined_at, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id)
		DO UPDATE SET
			username = excluded.username,
			updated_at = CURRENT_TIMESTAMP,
			joined_at = COALESCE(users.joined_at, excluded.joined_at)
	`, userID, username, joinedAt)

	return err
}
