package achievements

import (
	"brochievements/internal/storage"
)

func TopStreamTime(store *storage.Storage) (*WeeklyAchievement, error) {
	res, err := store.GetTopStreamTime()
	if err != nil || res == nil {
		return nil, nil
	}

	return &WeeklyAchievement{
		Code:        "top_stream_time",
		Title:       "🎥 В эфире больше всех",
		Description: "Провёл больше всех времени в стриме",
		UserID:      res.UserID,
		Username:    res.Username,
		Value:       formatDuration(res.Seconds),
	}, nil
}

func TopStreamViewer(store *storage.Storage) (*WeeklyAchievement, error) {
	res, err := store.GetTopStreamViewer()
	if err != nil || res == nil {
		return nil, nil
	}

	return &WeeklyAchievement{
		Code:        "top_stream_viewer",
		Title:       "👀 Главный зритель",
		Description: "Провёл больше всех времени, смотря чужие стримы",
		UserID:      res.UserID,
		Username:    res.Username,
		Value:       formatDuration(res.Seconds),
	}, nil
}
