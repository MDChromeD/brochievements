package storage

import (
	"database/sql"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	DB *sql.DB
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
    game TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS users (
    user_id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
	joined_at DATETIME,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS voice_channel_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    joined_at DATETIME NOT NULL,
    left_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS voice_channels (
    channel_id TEXT PRIMARY KEY,
    channel_name TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS stream_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    username TEXT,
    channel_id TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS stream_views (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    viewer_id TEXT NOT NULL,
    streamer_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS afk_game_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    username TEXT,
    game TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS achievement_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    username TEXT NOT NULL,
    achievement_code TEXT NOT NULL,
    achievement_title TEXT NOT NULL,
    value TEXT,
    description TEXT,
    awarded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    week_start DATE NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_achievement_user ON achievement_history(user_id);
	CREATE INDEX IF NOT EXISTS idx_achievement_code ON achievement_history(achievement_code);
	CREATE INDEX IF NOT EXISTS idx_achievement_week ON achievement_history(week_start);
	
	CREATE TABLE IF NOT EXISTS current_leaders (
    achievement_code TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    username TEXT NOT NULL,
    value TEXT NOT NULL,
    numeric_value REAL NOT NULL,
    checked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    week_start DATE NOT NULL
	);`

	if _, err := db.Exec(query); err != nil {
		log.Fatal(err)
	}

	return &Storage{DB: db}
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

type LongestVoiceSessionStat struct {
	UserID   string
	Username string
	Seconds  int
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
