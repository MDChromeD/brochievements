package achievements

import (
	"brochievements/internal/ai"
	"brochievements/internal/storage"
	"fmt"
	"log"
)

func collectWeeklyResults(store *storage.Storage) ([]WeeklyUserStats, error) {
	results, err := LoadWeeklyStats(store)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func PublishWeeklyDebug(store *storage.Storage, generator ai.Generator) {
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
		log.Printf("\n[weekly] %s | %s (%s) | value=%s",
			a.Title,
			a.Username,
			a.UserID,
			a.Value,
		)
		log.Printf("[weekly] Default description: %s", a.Description)

		if generator != nil {
			aiDesc, err := ai.GenerateAchievementText(generator, ai.AchievementAIContext{
				Title: a.Title,
				Kind:  "achievement",
				Fact:  fmt.Sprintf("%s получил достижение за %s", a.Username, a.Value),
			})
			if err != nil {
				log.Printf("[weekly][AI] ❌ Generation error: %v", err)
			} else {
				log.Printf("[weekly][AI] ✅ Generated: %s", aiDesc)
			}
		} else {
			log.Println("[weekly][AI] ⚠️ Generator disabled (OPENAI_API_KEY not set)")
		}
		log.Println("---")
	}

	log.Println("========== END WEEKLY DEBUG ==========")
}
