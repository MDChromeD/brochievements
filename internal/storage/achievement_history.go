package storage

import "time"

type AchievementRecord struct {
	ID               int64
	UserID           string
	Username         string
	AchievementCode  string
	AchievementTitle string
	Value            string
	Description      string
	AwardedAt        time.Time
	WeekStart        time.Time
}

// SaveAchievement сохраняет выданное достижение в историю
func (s *Storage) SaveAchievement(
	userID string,
	username string,
	code string,
	title string,
	value string,
	description string,
	weekStart time.Time,
) error {
	_, err := s.DB.Exec(`
		INSERT INTO achievement_history 
		(user_id, username, achievement_code, achievement_title, value, description, week_start)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, username, code, title, value, description, weekStart.Format("2006-01-02"))

	return err
}

// GetUserAchievements получает все достижения пользователя
func (s *Storage) GetUserAchievements(userID string) ([]AchievementRecord, error) {
	rows, err := s.DB.Query(`
		SELECT id, user_id, username, achievement_code, achievement_title, 
		       value, description, awarded_at, week_start
		FROM achievement_history
		WHERE user_id = ?
		ORDER BY awarded_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []AchievementRecord
	for rows.Next() {
		var r AchievementRecord
		err := rows.Scan(
			&r.ID,
			&r.UserID,
			&r.Username,
			&r.AchievementCode,
			&r.AchievementTitle,
			&r.Value,
			&r.Description,
			&r.AwardedAt,
			&r.WeekStart,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}

	return records, nil
}

// GetAchievementStats получает статистику по достижениям
func (s *Storage) GetAchievementStats(achievementCode string) ([]AchievementRecord, error) {
	rows, err := s.DB.Query(`
		SELECT id, user_id, username, achievement_code, achievement_title,
		       value, description, awarded_at, week_start
		FROM achievement_history
		WHERE achievement_code = ?
		ORDER BY awarded_at DESC
	`, achievementCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []AchievementRecord
	for rows.Next() {
		var r AchievementRecord
		err := rows.Scan(
			&r.ID,
			&r.UserID,
			&r.Username,
			&r.AchievementCode,
			&r.AchievementTitle,
			&r.Value,
			&r.Description,
			&r.AwardedAt,
			&r.WeekStart,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}

	return records, nil
}

// GetTopAchievementHolders получает топ пользователей по количеству достижений
func (s *Storage) GetTopAchievementHolders(limit int) ([]struct {
	UserID   string
	Username string
	Count    int
}, error) {
	rows, err := s.DB.Query(`
		SELECT user_id, username, COUNT(*) as count
		FROM achievement_history
		GROUP BY user_id
		ORDER BY count DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []struct {
		UserID   string
		Username string
		Count    int
	}

	for rows.Next() {
		var r struct {
			UserID   string
			Username string
			Count    int
		}
		if err := rows.Scan(&r.UserID, &r.Username, &r.Count); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}
