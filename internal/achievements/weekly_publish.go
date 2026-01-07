package achievements

import (
	"log"

	"brochievements/internal/ai"
	"brochievements/internal/storage"

	"github.com/bwmarrin/discordgo"
)

func PublishWeeklyDebug(
	s *discordgo.Session,
	store *storage.Storage,
	channelID string,
	generator ai.Generator,
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

		ctx := buildAIContext(a)

		text, err := ai.GenerateAchievementText(generator, ctx)
		if err != nil {
			text = "[AI ERROR] " + err.Error()
		}

		log.Printf(
			"[weekly]\nTitle: %s\nUser: %s\nValue: %s\nAI: %s\n",
			a.Title,
			a.Username,
			a.Value,
			text,
		)
	}

	// 5️⃣ публикуем в Discord
	if channelID != "" {
		// sendWeeklyEmbed(s, channelID, results)
		// sendWeeklyEmbedWithAI(s, channelID, results, generator)
	}

	log.Println("[weekly] END weekly debug publish")
}

func sendWeeklyEmbedWithAI(
	s *discordgo.Session,
	channelID string,
	results []*WeeklyAchievement,
	generator ai.Generator,
) {
	embed := &discordgo.MessageEmbed{
		Title:       "📊 Статистика достижений",
		Description: "Срез активности на момент запуска бота",
		Color:       0x5865F2,
	}

	for _, a := range results {

		// fallback — старое описание
		text := a.Description

		// AI-текст, если доступен
		if generator != nil {
			ctx := buildAIContext(a)
			if t, err := ai.GenerateAchievementText(generator, ctx); err == nil {
				text = t
			}
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  a.Title + " — " + a.Username,
			Value: text + "\n**" + a.Value + "**",
		})
	}

	if _, err := s.ChannelMessageSendEmbed(channelID, embed); err != nil {
		log.Println("[weekly] send embed error:", err)
	}
}

func buildAIContext(a *WeeklyAchievement) ai.AchievementAIContext {
	switch a.Code {

	case "port_whore":
		return ai.AchievementAIContext{
			Title: "Портовая шлюха",
			Kind:  "achievement",
			Fact:  "пользователь чаще всех менял игры за неделю",
		}

	case "lovebirds":
		return ai.AchievementAIContext{
			Title: "Голубки",
			Kind:  "achievement",
			Fact:  "два пользователя чаще всего сидели вместе в одном голосовом канале",
		}

	case "afk":
		return ai.AchievementAIContext{
			Title: "AFK",
			Kind:  "anti-achievement",
			Fact:  "пользователь чаще остальных бездействовал",
		}

	default:
		return ai.AchievementAIContext{
			Title: a.Title,
			Kind:  "achievement",
			Fact:  "пользователь отличился активностью на сервере",
		}
	}
}
