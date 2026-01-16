package achievements

import (
	"brochievements/internal/storage"
)

func AFKFarmer(store *storage.Storage) (*WeeklyAchievement, error) {
	res, err := store.GetTopAFKFarmer()
	if err != nil || res == nil || res.AFKSeconds == 0 {
		return nil, nil
	}

	return &WeeklyAchievement{
		Code:        "afk_farmer",
		Title:       "🥔 Фермер AFK",
		Description: "Больше всех времени провёл AFK с запущенной игрой",
		UserID:      res.UserID,
		Username:    res.Username,
		Value:       formatDuration(res.AFKSeconds),
	}, nil
}
