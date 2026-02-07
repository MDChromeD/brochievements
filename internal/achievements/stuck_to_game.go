package achievements

import "fmt"

func StuckToGame(stats []WeeklyUserStats) *WeeklyAchievement {
	var winner *WeeklyUserStats

	for i := range stats {
		s := &stats[i]

		if s.MaxSingleGameSec == 0 {
			continue
		}

		if winner == nil || s.MaxSingleGameSec > winner.MaxSingleGameSec {
			winner = s
		}
	}

	if winner == nil || winner.MaxSingleGameSec < 3600 {
		return nil
	}

	return &WeeklyAchievement{
		Code:        "stuck_to_game",
		Title:       "🍌 Застрял на бибе",
		Description: "Дольше всех играл в одну и ту же игру",
		UserID:      winner.UserID,
		Username:    winner.Username,
		Value: fmt.Sprintf(
			"%s — %.1f ч",
			winner.MaxSingleGame,
			float64(winner.MaxSingleGameSec)/3600,
		),
		Kind: "achievement",
	}
}
