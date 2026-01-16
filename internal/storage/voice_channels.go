package storage

func (s *Storage) UpsertVoiceChannel(channelID, name string) error {
	_, err := s.DB.Exec(`
		INSERT INTO voice_channels (channel_id, name)
		VALUES (?, ?)
		ON CONFLICT(channel_id) DO UPDATE SET
			name = excluded.name
	`, channelID, name)
	return err
}

func (s *Storage) GetVoiceChannelName(channelID string) (string, error) {
	row := s.DB.QueryRow(`
		SELECT name
		FROM voice_channels
		WHERE channel_id = ?
	`, channelID)

	var name string
	err := row.Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}
