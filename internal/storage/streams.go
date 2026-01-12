package storage

import (
	"database/sql"
)

// ---------- STREAM SESSIONS ----------

func (s *Storage) StartStreamSession(userID, username, channelID string) error {
	// защита от дублей: если уже есть активный стрим — ничего не делаем
	var exists int
	err := s.DB.QueryRow(`
		SELECT 1
		FROM stream_sessions
		WHERE user_id = ? AND ended_at IS NULL
		LIMIT 1
	`, userID).Scan(&exists)

	if err == nil {
		return nil // уже стримит
	}
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	_, err = s.DB.Exec(`
		INSERT INTO stream_sessions (user_id, username, channel_id, started_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`, userID, username, channelID)
	return err
}

func (s *Storage) EndStreamSession(userID string) error {
	_, err := s.DB.Exec(`
		UPDATE stream_sessions
		SET ended_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND ended_at IS NULL
	`, userID)
	return err
}

func (s *Storage) CloseUnfinishedStreamSessions() error {
	_, err := s.DB.Exec(`
		UPDATE stream_sessions
		SET ended_at = CURRENT_TIMESTAMP
		WHERE ended_at IS NULL
	`)
	return err
}

func (s *Storage) GetActiveStreamers(channelID string) ([]string, error) {
	rows, err := s.DB.Query(`
		SELECT user_id
		FROM stream_sessions
		WHERE channel_id = ? AND ended_at IS NULL
	`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// -- STREAM VIEWS --

func (s *Storage) StartStreamView(viewerID, streamerID, channelID string) error {
	// защита от дублей
	var exists int
	err := s.DB.QueryRow(`
		SELECT 1
		FROM stream_views
		WHERE viewer_id = ? AND streamer_id = ? AND channel_id = ? AND ended_at IS NULL
		LIMIT 1
	`, viewerID, streamerID, channelID).Scan(&exists)

	if err == nil {
		return nil // уже "смотрит"
	}
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	_, err = s.DB.Exec(`
		INSERT INTO stream_views (viewer_id, streamer_id, channel_id, started_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`, viewerID, streamerID, channelID)
	return err
}

func (s *Storage) EndStreamView(viewerID, streamerID, channelID string) error {
	_, err := s.DB.Exec(`
		UPDATE stream_views
		SET ended_at = CURRENT_TIMESTAMP
		WHERE viewer_id = ? AND streamer_id = ? AND channel_id = ? AND ended_at IS NULL
	`, viewerID, streamerID, channelID)
	return err
}

func (s *Storage) EndAllViewsForStreamer(streamerID string) error {
	_, err := s.DB.Exec(`
		UPDATE stream_views
		SET ended_at = CURRENT_TIMESTAMP
		WHERE streamer_id = ? AND ended_at IS NULL
	`, streamerID)
	return err
}

func (s *Storage) EndAllViewsForViewer(viewerID string) error {
	_, err := s.DB.Exec(`
		UPDATE stream_views
		SET ended_at = CURRENT_TIMESTAMP
		WHERE viewer_id = ? AND ended_at IS NULL
	`, viewerID)
	return err
}

func (s *Storage) EndAllViewsForViewerInChannel(viewerID, channelID string) error {
	_, err := s.DB.Exec(`
		UPDATE stream_views
		SET ended_at = CURRENT_TIMESTAMP
		WHERE viewer_id = ? AND channel_id = ? AND ended_at IS NULL
	`, viewerID, channelID)
	return err
}

func (s *Storage) CloseUnfinishedStreamViews() error {
	_, err := s.DB.Exec(`
		UPDATE stream_views
		SET ended_at = CURRENT_TIMESTAMP
		WHERE ended_at IS NULL
	`)
	return err
}

// Sync: привести просмотры пользователя в канале к актуальному состоянию
func (s *Storage) SyncStreamViews(viewerID, channelID string, viewerIsStreaming bool, viewerIsDeaf bool) error {
	// если пользователь сам стримит или заглушился — он не зритель
	if viewerIsStreaming || viewerIsDeaf || channelID == "" {
		return s.EndAllViewsForViewerInChannel(viewerID, channelID)
	}

	streamers, err := s.GetActiveStreamers(channelID)
	if err != nil {
		return err
	}

	// если никто не стримит — закрываем просмотры
	if len(streamers) == 0 {
		return s.EndAllViewsForViewerInChannel(viewerID, channelID)
	}

	// 1) Закрываем просмотры "лишних" стримеров (которых уже нет)
	_, err = s.DB.Exec(`
		UPDATE stream_views
		SET ended_at = CURRENT_TIMESTAMP
		WHERE viewer_id = ? AND channel_id = ? AND ended_at IS NULL
		  AND streamer_id NOT IN (
			SELECT user_id
			FROM stream_sessions
			WHERE channel_id = ? AND ended_at IS NULL
		  )
	`, viewerID, channelID, channelID)
	if err != nil {
		return err
	}

	// 2) Открываем просмотры для всех активных стримеров (кроме самого viewer)
	for _, streamerID := range streamers {
		if streamerID == viewerID {
			continue
		}
		if err := s.StartStreamView(viewerID, streamerID, channelID); err != nil {
			return err
		}
	}

	return nil
}
