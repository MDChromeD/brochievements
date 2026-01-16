package achievements

import (
	"brochievements/internal/storage"
	"log"
)

func collectWeeklyResults(store *storage.Storage) ([]WeeklyUserStats, error) {
	results, err := LoadWeeklyStats(store)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func PublishWeeklyDebug(store *storage.Storage) {
	log.Println("========== WEEKLY ACHIEVEMENTS DEBUG ==========")

	results, err := collectWeeklyResults(store)
	if err != nil {
		log.Println("[weekly] collect error:", err)
		return
	}

	if len(results) == 0 {
		log.Println("[weekly] no achievements calculated")
		return
	}

	stats, err := LoadWeeklyStats(store)
	if err != nil {
		log.Println("[weekly] load stats error:", err)
		return
	}

	achievements := RunWeeklyAll(stats)

	if len(achievements) == 0 {
		log.Println("[weekly] no achievements this week")
		return
	}

	for _, a := range achievements {
		// ВАЖНО: не ставь лишние %s — иначе снова будет %!s(MISSING)
		log.Printf("[weekly] %s | %s (%s) | value=%s",
			a.Title,
			a.Username,
			a.UserID,
			a.Value,
		)
	}

	log.Println("========== END WEEKLY DEBUG ==========")
}
