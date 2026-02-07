package achievements

import "fmt"

func CasualGamer(stats []WeeklyUserStats) *WeeklyAchievement {
	var loser *WeeklyUserStats

	for i := range stats {
		s := &stats[i]

		// игнорим тех, кто вообще не играл
		if s.GameSeconds == 0 {
			continue
		}

		// ищем минимум
		if loser == nil || s.GameSeconds < loser.GameSeconds {
			loser = s
		}
	}

	// никто не заслужил (все либо много играли, либо вообще не играли)
	//if loser == nil || loser.GameSeconds > 7200 { // > 2 часов
	if loser == nil { // > 2 часов
		return nil
	}

	return &WeeklyAchievement{
		Code:        "casual_gamer",
		Title:       "🎮 Казуал",
		Description: "Играл на неделе так мало, что мог бы и не появляться.",
		UserID:      loser.UserID,
		Username:    loser.Username,
		Value:       fmt.Sprintf("%.1f ч", float64(loser.GameSeconds)/3600),
		Kind:        "anti-achievement",
	}
}
