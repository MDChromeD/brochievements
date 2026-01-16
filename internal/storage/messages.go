package storage

func (s *Storage) SaveMessage(
	userID string,
	username string,
	channelID string,
	content string,
) error {
	_, err := s.DB.Exec(`
		INSERT INTO messages (user_id, username, channel_id, content)
		VALUES (?, ?, ?, ?)
	`, userID, username, channelID, content)
	return err
}
