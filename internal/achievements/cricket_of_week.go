package achievements

import "fmt"

func CricketOfWeek(stats []WeeklyUserStats) *WeeklyAchievement {
	var loser *WeeklyUserStats

	for i := range stats {
		s := &stats[i]

		// игнорим тех, кто вообще не заходил
		if s.VoiceSeconds == 0 {
			continue
		}

		// ищем минимум
		if loser == nil || s.VoiceSeconds < loser.VoiceSeconds {
			loser = s
		}
	}

	// никто не заслужил (все либо много сидели, либо вообще не заходили)
	//if loser == nil || loser.VoiceSeconds > 1800 { // > 30 минут
	if loser == nil { // > 30 минут
		return nil
	}

	return &WeeklyAchievement{
		Code:        "cricket_of_week",
		Title:       "🦗 Прозрачный чиркаш",
		Description: "Провел в голосе меньше всех времени на неделе",
		UserID:      loser.UserID,
		Username:    loser.Username,
		Value:       fmt.Sprintf("%.1f мин", float64(loser.VoiceSeconds)/60),
		Kind:        "anti-achievement",
	}
}
