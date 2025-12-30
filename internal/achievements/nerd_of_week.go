package achievements

import "fmt"

func NerdOfWeek(stats []WeeklyUserStats) *WeeklyAchievement {
	var winner *WeeklyUserStats

	for i := range stats {
		s := &stats[i]

		if s.VoiceSeconds == 0 {
			continue
		}

		if winner == nil || s.VoiceSeconds > winner.VoiceSeconds {
			winner = s
		}
	}

	if winner == nil || winner.VoiceSeconds < 7200 {
		return nil
	}

	return &WeeklyAchievement{
		Code:        "nerd_of_week",
		Title:       "(_*_) На все щели мастер",
		Description: "Провёл больше всех времени в голосе",
		UserID:      winner.UserID,
		Username:    winner.Username,
		Value:       fmt.Sprintf("%.1f ч", float64(winner.VoiceSeconds)/3600),
	}
}
