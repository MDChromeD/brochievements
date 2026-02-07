package storage

import (
	"database/sql"
	"time"
)

type CurrentLeader struct {
	AchievementCode string
	UserID          string
	Username        string
	Value           string
	NumericValue    float64
	CheckedAt       time.Time
	WeekStart       time.Time
}

// GetCurrentLeader получает текущего лидера по коду достижения
func (s *Storage) GetCurrentLeader(achievementCode string) (*CurrentLeader, error) {
	row := s.DB.QueryRow(`
		SELECT achievement_code, user_id, username, value, numeric_value, checked_at, week_start
		FROM current_leaders
		WHERE achievement_code = ?
	`, achievementCode)

	var leader CurrentLeader
	err := row.Scan(
		&leader.AchievementCode,
		&leader.UserID,
		&leader.Username,
		&leader.Value,
		&leader.NumericValue,
		&leader.CheckedAt,
		&leader.WeekStart,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &leader, nil
}

// UpdateCurrentLeader обновляет или создаёт запись текущего лидера
func (s *Storage) UpdateCurrentLeader(
	achievementCode string,
	userID string,
	username string,
	value string,
	numericValue float64,
	weekStart time.Time,
) error {
	_, err := s.DB.Exec(`
		INSERT INTO current_leaders (achievement_code, user_id, username, value, numeric_value, week_start)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(achievement_code) DO UPDATE SET
			user_id = excluded.user_id,
			username = excluded.username,
			value = excluded.value,
			numeric_value = excluded.numeric_value,
			checked_at = CURRENT_TIMESTAMP,
			week_start = excluded.week_start
	`, achievementCode, userID, username, value, numericValue, weekStart.Format("2006-01-02"))

	return err
}

// ClearOldLeaders удаляет лидеров старых недель
func (s *Storage) ClearOldLeaders(weekStart time.Time) error {
	_, err := s.DB.Exec(`
		DELETE FROM current_leaders
		WHERE week_start < ?
	`, weekStart.Format("2006-01-02"))

	return err
}

// GetAllCurrentLeaders получает всех текущих лидеров
func (s *Storage) GetAllCurrentLeaders() ([]CurrentLeader, error) {
	rows, err := s.DB.Query(`
		SELECT achievement_code, user_id, username, value, numeric_value, checked_at, week_start
		FROM current_leaders
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaders []CurrentLeader
	for rows.Next() {
		var leader CurrentLeader
		err := rows.Scan(
			&leader.AchievementCode,
			&leader.UserID,
			&leader.Username,
			&leader.Value,
			&leader.NumericValue,
			&leader.CheckedAt,
			&leader.WeekStart,
		)
		if err != nil {
			return nil, err
		}
		leaders = append(leaders, leader)
	}

	return leaders, nil
}
