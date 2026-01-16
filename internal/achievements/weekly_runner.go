package achievements

func RunWeeklyAll(stats []WeeklyUserStats) []*WeeklyAchievement {
	var result []*WeeklyAchievement

	// 🏆 обычные
	if a := PortWhore(stats); a != nil {
		result = append(result, a)
	}
	if a := StuckToGame(stats); a != nil {
		result = append(result, a)
	}
	if a := WentForBread(stats); a != nil {
		result = append(result, a)
	}
	if a := NerdOfWeek(stats); a != nil {
		result = append(result, a)
	}

	return result
}
