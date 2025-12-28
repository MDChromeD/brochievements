package storage

import (
	"database/sql"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	DB *sql.DB
}

type GameSession struct {
	ID   int64
	Game string
}

func New(path string) *Storage {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal(err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT,
		username TEXT,
		channel_id TEXT,
		content TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS voice_sessions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id TEXT,
	username TEXT,
	channel_id TEXT,
	joined_at DATETIME,
	left_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS game_sessions (
  	id INTEGER PRIMARY KEY AUTOINCREMENT,
  	user_id TEXT NOT NULL,
  	username TEXT NOT NULL,
  	game TEXT NOT NULL,
  	started_at DATETIME NOT NULL,
  	ended_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS users (
    user_id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(query); err != nil {
		log.Fatal(err)
	}

	return &Storage{DB: db}
}

func (s *Storage) SaveMessage(
	userID string,
	username string,
	channelID string,
	content string,
) error {
	_, err := s.DB.Exec(
		`INSERT INTO messages (user_id, username, channel_id, content)
		 VALUES (?, ?, ?, ?)`,
		userID,
		username,
		channelID,
		content,
	)
	return err
}

func (s *Storage) StartVoiceSession(
	userID, username, channelID string,
) error {
	_, err := s.DB.Exec(`
		INSERT INTO voice_sessions (user_id, username, channel_id, joined_at)
		VALUES (?, ?, ?, datetime('now'))
	`, userID, username, channelID)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil // сессия уже активна → игнор
		}
		return err
	}
	return nil
}

func (s *Storage) EndVoiceSession(
	userID string,
) error {
	_, err := s.DB.Exec(`
		UPDATE voice_sessions
		SET left_at = datetime('now')
		WHERE user_id = ?
		  AND left_at IS NULL
	`, userID)

	return err
}

type VoiceTimeStat struct {
	UserID   string
	Username string
	Seconds  int
}

func (s *Storage) TopVoiceUserLastWeek() (*VoiceTimeStat, error) {
	row := s.DB.QueryRow(`
		SELECT
			user_id,
			username,
			SUM(
				strftime('%s', COALESCE(left_at, datetime('now')))
				- strftime('%s', joined_at)
			) as seconds
		FROM voice_sessions
		WHERE joined_at >= datetime('now', '-7 days')
		GROUP BY user_id, username
		ORDER BY seconds DESC
		LIMIT 1
	`)

	var stat VoiceTimeStat
	err := row.Scan(&stat.UserID, &stat.Username, &stat.Seconds)
	if err != nil {
		return nil, err
	}

	return &stat, nil
}

type VoiceJoinStat struct {
	UserID   string
	Username string
	Count    int
}

type GameStat struct {
	Username string
	Game     string
	Count    int
}

func (s *Storage) UpsertUser(userID, username string) error {
	if username == "" {
		return nil
	}

	_, err := s.DB.Exec(`
		INSERT INTO users (user_id, username, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id)
		DO UPDATE SET
			username = excluded.username,
			updated_at = CURRENT_TIMESTAMP
	`, userID, username)

	return err
}

func (s *Storage) TopVoiceJoinsLastWeek() (*VoiceJoinStat, error) {
	row := s.DB.QueryRow(`
		SELECT user_id, username, COUNT(*) as count
		FROM voice_sessions
		WHERE joined_at >= datetime('now', '-7 days')
		GROUP BY user_id, username
		ORDER BY count DESC
		LIMIT 1
	`)

	var stat VoiceJoinStat
	err := row.Scan(&stat.UserID, &stat.Username, &stat.Count)
	if err != nil {
		return nil, err
	}

	return &stat, nil
}

type LongestVoiceSessionStat struct {
	UserID   string
	Username string
	Seconds  int
}

func (s *Storage) LongestVoiceSessionLastWeek() (*LongestVoiceSessionStat, error) {
	row := s.DB.QueryRow(`
		SELECT
			user_id,
			username,
			MAX(
				strftime('%s', COALESCE(left_at, datetime('now')))
				- strftime('%s', joined_at)
			) AS seconds
		FROM voice_sessions
		WHERE joined_at >= datetime('now', '-7 days')
		GROUP BY user_id, username
		ORDER BY seconds DESC
		LIMIT 1
	`)

	var stat LongestVoiceSessionStat
	err := row.Scan(&stat.UserID, &stat.Username, &stat.Seconds)
	if err != nil {
		return nil, err
	}

	return &stat, nil
}

type UserStats struct {
	MessagesCount int
	VoiceSeconds  int64
	GamesCount    int
	FirstSeen     time.Time
}

func (s *Storage) CountMessages(userID string) (int, error) {
	var count int
	err := s.DB.QueryRow(`
		SELECT COUNT(*)
		FROM messages
		WHERE user_id = ?
	`, userID).Scan(&count)

	return count, err
}

func (s *Storage) VoiceTimeSeconds(userID string) (int64, error) {
	var seconds int64
	err := s.DB.QueryRow(`
		SELECT COALESCE(SUM(
			(strftime('%s', left_at) - strftime('%s', joined_at))
		), 0)
		FROM voice_sessions
		WHERE user_id = ?
		  AND left_at IS NOT NULL
	`, userID).Scan(&seconds)

	return seconds, err
}

func (s *Storage) FirstSeen(userID string) (time.Time, error) {
	var ts string
	err := s.DB.QueryRow(`
		SELECT MIN(created_at)
		FROM messages
		WHERE user_id = ?
	`, userID).Scan(&ts)

	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(time.RFC3339, ts)
}

func (s *Storage) GetActiveGameSession(userID string) (*GameSession, error) {
	row := s.DB.QueryRow(`
		SELECT id, game
		FROM game_sessions
		WHERE user_id = ? AND ended_at IS NULL
		LIMIT 1
	`, userID)

	var gs GameSession
	err := row.Scan(&gs.ID, &gs.Game)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &gs, nil
}
