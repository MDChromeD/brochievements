package storage

import "database/sql"

type VoiceChannelSession struct {
	ID        int64
	ChannelID string
}

func (s *Storage) GetActiveVoiceChannelSession(userID string) (*VoiceChannelSession, error) {
	row := s.DB.QueryRow(`
		SELECT id, channel_id
		FROM voice_channel_sessions
		WHERE user_id = ? AND left_at IS NULL
		LIMIT 1
	`, userID)

	var vcs VoiceChannelSession
	err := row.Scan(&vcs.ID, &vcs.ChannelID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &vcs, nil
}

func (s *Storage) StartVoiceChannelSession(userID, channelID string) error {
	_, err := s.DB.Exec(`
		INSERT INTO voice_channel_sessions (user_id, channel_id, joined_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`, userID, channelID)

	return err
}

func (s *Storage) EndVoiceChannelSession(sessionID int64) error {
	_, err := s.DB.Exec(`
		UPDATE voice_channel_sessions
		SET left_at = CURRENT_TIMESTAMP
		WHERE id = ? AND left_at IS NULL
	`, sessionID)

	return err
}

func (s *Storage) EnsureVoiceChannelSession(userID, channelID string) error {
	active, err := s.GetActiveVoiceChannelSession(userID)
	if err != nil {
		return err
	}

	// ▶ нет активной — стартуем
	if active == nil {
		return s.StartVoiceChannelSession(userID, channelID)
	}

	// 🔁 тот же канал — ничего не делаем
	if active.ChannelID == channelID {
		return nil
	}

	// 🔄 канал сменился
	if err := s.EndVoiceChannelSession(active.ID); err != nil {
		return err
	}

	return s.StartVoiceChannelSession(userID, channelID)
}
