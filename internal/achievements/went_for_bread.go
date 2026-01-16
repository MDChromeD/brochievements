package achievements

import "fmt"

func WentForBread(stats []WeeklyUserStats) *WeeklyAchievement {
	var winner *WeeklyUserStats

	for i := range stats {
		s := &stats[i]

		if s.AFKSeconds == 0 {
			continue
		}

		if winner == nil || s.AFKSeconds > winner.AFKSeconds {
			winner = s
		}
	}

	if winner == nil || winner.AFKSeconds < 1800 {
		return nil
	}

	return &WeeklyAchievement{
		Code:        "went_for_bread",
		Title:       "💤 Ушёл за хлебушком",
		Description: "Провёл больше всех времени в состоянии away from",
		UserID:      winner.UserID,
		Username:    winner.Username,
		Value:       fmt.Sprintf("%.1f ч", float64(winner.AFKSeconds)/3600),
	}
}
