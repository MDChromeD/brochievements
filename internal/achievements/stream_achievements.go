package achievements

import "fmt"

func TopStreamer(stats []WeeklyUserStats) *WeeklyAchievement {
	var winner *WeeklyUserStats

	for i := range stats {
		s := &stats[i]

		if s.StreamSeconds == 0 {
			continue
		}

		if winner == nil || s.StreamSeconds > winner.StreamSeconds {
			winner = s
		}
	}

	if winner == nil || winner.StreamSeconds < 3600 {
		return nil
	}

	return &WeeklyAchievement{
		Code:        "top_stream_time",
		Title:       "🎥 В эфире больше всех",
		Description: "Провёл больше всех времени с включенным стримом",
		UserID:      winner.UserID,
		Username:    winner.Username,
		Value:       fmt.Sprintf("%.1f ч", float64(winner.StreamSeconds)/3600),
		Kind:        "achievement",
	}

}

func TopViewer(stats []WeeklyUserStats) *WeeklyAchievement {
	var winner *WeeklyUserStats

	for i := range stats {
		s := &stats[i]

		if s.StreamViewSeconds == 0 {
			continue
		}

		if winner == nil || s.StreamViewSeconds > winner.StreamViewSeconds {
			winner = s
		}
	}

	// if winner == nil || winner.StreamViewSeconds < 3600 {
	// 	return nil
	// }

	return &WeeklyAchievement{
		Code:        "top_stream_viewer",
		Title:       "👀 Главный зритель",
		Description: "Провёл больше всех времени, смотря чужие стримы",
		UserID:      winner.UserID,
		Username:    winner.Username,
		Value:       fmt.Sprintf("%.1f ч", float64(winner.StreamViewSeconds)/3600),
		Kind:        "achievement",
	}

}
