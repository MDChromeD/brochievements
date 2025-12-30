package achievements

import "fmt"

func Lovebirds(pair *DovePair) *WeeklyAchievement {
	if pair == nil {
		return nil
	}

	return &WeeklyAchievement{
		Code:        "lovebirds",
		Title:       "🕊 Голубки",
		Description: "Чаще всех сидели наедине в одном голосовом канале",
		UserID:      pair.UserA + "+" + pair.UserB,
		Username:    fmt.Sprintf("%s ❤️ %s", pair.NameA, pair.NameB),
		Value:       fmt.Sprintf("%.1f ч вместе", float64(pair.Seconds)/3600),
	}
}
