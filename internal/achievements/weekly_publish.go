package achievements

import (
	"log"

	"brochievements/internal/storage"

	"github.com/bwmarrin/discordgo"
)

func PublishWeeklyDebug(
	s *discordgo.Session,
	store *storage.Storage,
	channelID string,
) {
	log.Println("[weekly] START weekly debug publish")

	// 1️⃣ грузим weekly-стату
	stats, err := LoadWeeklyStats(store)
	if err != nil {
		log.Println("[weekly] LoadWeeklyStats error:", err)
		return
	}

	if err := LoadWeeklyAFK(store, stats); err != nil {
		log.Println("[weekly] LoadWeeklyAFK error:", err)
		return
	}

	if err := LoadWeeklyMaxGame(store, stats); err != nil {
		log.Println("[weekly] LoadWeeklyMaxGame error:", err)
		return
	}

	log.Printf("[weekly] users loaded: %d", len(stats))

	// 2️⃣ обычные + анти
	results := RunWeeklyAll(stats)

	// 3️⃣ голубки
	doves, err := LoadWeeklyDoves(store)
	if err != nil {
		log.Println("[weekly] LoadWeeklyDoves error:", err)
	} else if a := Lovebirds(doves); a != nil {
		results = append(results, a)
	}

	// 4️⃣ логируем ВСЁ
	log.Println("===== WEEKLY RESULTS (DEBUG) =====")
	if len(results) == 0 {
		log.Println("[weekly] nobody deserved anything 😈")
	}

	for _, a := range results {
		log.Printf(
			"[weekly] %s → %s (%s)",
			a.Title,
			a.Username,
			a.Value,
		)
	}

	// 5️⃣ публикуем в Discord
	//if channelID != "" {
	//	sendWeeklyEmbed(s, channelID, results)
	//}

	log.Println("[weekly] END weekly debug publish")
}
