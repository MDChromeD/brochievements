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

func (s *Storage) fillMessageStats(userID string, stats *UserStats) {
	s.DB.QueryRow(
		`SELECT COUNT(*) FROM messages WHERE user_id = ?`,
		userID,
	).Scan(&stats.MessagesCount)

	s.DB.QueryRow(
		`SELECT joined_at FROM users WHERE user_id = ?`,
		userID,
	).Scan(&stats.JoinedAt)
}

func (s *Storage) fillVoiceStats(userID string, stats *UserStats) {
	s.DB.QueryRow(`
		SELECT 
			IFNULL(SUM(strftime('%s', left_at) - strftime('%s', joined_at)), 0),
			COUNT(*)
		FROM voice_sessions
		WHERE user_id = ? AND left_at IS NOT NULL
	`, userID).Scan(&stats.VoiceSeconds, &stats.VoiceJoins)
}

func (s *Storage) fillFavoriteChannel(userID string, stats *UserStats) {
	s.DB.QueryRow(`
		SELECT vc.channel_name,
		       SUM(strftime('%s', vcs.left_at) - strftime('%s', vcs.joined_at)) AS seconds
		FROM voice_channel_sessions vcs
		JOIN voice_channels vc ON vc.channel_id = vcs.channel_id
		WHERE vcs.user_id = ? AND vcs.left_at IS NOT NULL
		GROUP BY vcs.channel_id
		ORDER BY seconds DESC
		LIMIT 1
	`, userID).Scan(&stats.FavoriteChannel, &stats.FavoriteChannelSec)
}

func (s *Storage) fillGameStats(userID string, stats *UserStats) {
	s.DB.QueryRow(`
		SELECT IFNULL(SUM(strftime('%s', ended_at) - strftime('%s', started_at)), 0)
		FROM game_sessions
		WHERE user_id = ? AND ended_at IS NOT NULL
	`, userID).Scan(&stats.GameSeconds)

	s.DB.QueryRow(`
		SELECT game,
		       SUM(strftime('%s', ended_at) - strftime('%s', started_at)) AS seconds
		FROM game_sessions
		WHERE user_id = ? AND ended_at IS NOT NULL
		GROUP BY game
		ORDER BY seconds DESC
		LIMIT 1
	`, userID).Scan(&stats.FavoriteGame, &stats.FavoriteGameSec)

	s.DB.QueryRow(`
		SELECT COUNT(*) FROM game_sessions WHERE user_id = ?
	`, userID).Scan(&stats.GameSwitches)
}

func (s *Storage) UpsertVoiceChannel(channelID, channelName string) error {
	if channelID == "" || channelName == "" {
		return nil
	}

	_, err := s.DB.Exec(`
		INSERT INTO voice_channels (channel_id, channel_name, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(channel_id)
		DO UPDATE SET
			channel_name = excluded.channel_name,
			updated_at = CURRENT_TIMESTAMP
	`, channelID, channelName)

	return err
}

func (s *Storage) GetVoiceChannelName(channelID string) (string, bool) {
	row := s.DB.QueryRow(`
		SELECT channel_name
		FROM voice_channels
		WHERE channel_id = ?
	`, channelID)

	var name string
	err := row.Scan(&name)
	if err != nil {
		return "", false
	}

	return name, true
}
