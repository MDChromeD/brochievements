package achievements

import "fmt"

func PortWhore(stats []WeeklyUserStats) *WeeklyAchievement {
	var winner *WeeklyUserStats

	for i := range stats {
		s := &stats[i]

		// игнорим тех, кто вообще не играл
		if s.GameSessionsCount == 0 {
			continue
		}

		if winner == nil || s.GameSessionsCount > winner.GameSessionsCount {
			winner = s
		}
	}

	// никто не заслужил
	if winner == nil || winner.GameSessionsCount < 2 {
		return nil
	}

	return &WeeklyAchievement{
		Code:        "port_whore",
		Title:       "🔁 Портовая шлюха",
		Description: "Менял игры чаще всех за неделю",
		UserID:      winner.UserID,
		Username:    winner.Username,
		Value:       fmt.Sprintf("%d смен игр", winner.GameSessionsCount),
	}
}
