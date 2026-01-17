package achievements

import (
	"brochievements/internal/storage"
	"time"
)

func lastWeekRange() (from, to time.Time) {
	now := time.Now()
	to = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	from = to.AddDate(0, 0, -7)
	return
}

func LoadWeeklyStats(store *storage.Storage) ([]WeeklyUserStats, error) {
	from, to := lastWeekRange()

	voiceRows, err := store.LoadWeeklyVoiceRaw(from, to)
	if err != nil {
		return nil, err
	}

	gameRows, err := store.LoadWeeklyGameRaw(from, to)
	if err != nil {
		return nil, err
	}

	afkGameRows, err := store.LoadWeeklyAFKGameRaw(from, to)
	if err != nil {
		return nil, err
	}

	afkRows, err := store.LoadWeeklyBreadAFKRaw(from, to, "Я_за_хлебушком_пацаны")
	if err != nil {
		return nil, err
	}

	StreamRows, err := store.LoadWeeklyStreamRaw(from, to)
	if err != nil {
		return nil, err
	}

	StreamViewRows, err := store.LoadWeeklyStreamViewRaw(from, to)
	if err != nil {
		return nil, err
	}

	byUser := map[string]*WeeklyUserStats{}

	get := func(userID, username string) *WeeklyUserStats {
		st := byUser[userID]
		if st == nil {
			st = &WeeklyUserStats{UserID: userID}
			byUser[userID] = st
		}
		if st.Username == "" && username != "" {
			st.Username = username
		}
		return st
	}

	for _, r := range voiceRows {
		st := get(r.UserID, r.Username)
		st.VoiceSeconds = r.Seconds
	}

	for _, r := range afkGameRows {
		st := get(r.UserID, r.Username)
		st.AfkGameSeconds = r.Seconds
	}

	for _, r := range afkRows {
		st := get(r.UserID, r.Username)
		st.AFKSeconds = r.Seconds
	}

	for _, r := range StreamRows {
		st := get(r.UserID, r.Username)
		st.StreamSeconds = r.Seconds
	}

	for _, r := range StreamViewRows {
		st := get(r.UserID, r.Username)
		st.StreamViewSeconds = r.Seconds
	}

	for _, r := range gameRows {
		st := get(r.UserID, r.Username)
		st.GameSeconds += r.Seconds
		st.DistinctGames++

		if st.MaxSingleGameSec < r.Seconds {
			st.MaxSingleGameSec = r.Seconds
			st.MaxSingleGame = r.Game
		}
	}

	res := make([]WeeklyUserStats, 0, len(byUser))
	for _, st := range byUser {
		if st.Username == "" {
			st.Username = st.UserID
		}
		res = append(res, *st)
	}
	return res, nil
}
