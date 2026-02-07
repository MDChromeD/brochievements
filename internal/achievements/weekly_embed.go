package achievements

import (
	"brochievements/internal/ai"
	"brochievements/internal/storage"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// SendWeeklyEmbed отправляет embed с еженедельными достижениями
func SendWeeklyEmbed(
	s *discordgo.Session,
	channelID string,
	achs []*WeeklyAchievement,
	generator ai.Generator,
	store *storage.Storage,
) {
	log.Println("[weekly_embed] === START SendWeeklyEmbed ===")

	if len(achs) == 0 {
		log.Println("[weekly_embed] No achievements to publish")
		s.ChannelMessageSend(
			channelID,
			"📊 Итоги недели\n\nНикто ничего не заслужил 😈",
		)
		return
	}

	// 🔍 ДИАГНОСТИКА GENERATOR
	if generator == nil {
		log.Println("[weekly_embed] ⚠️ AI Generator is NIL!")
	} else {
		log.Println("[weekly_embed] ✅ AI Generator is available")
	}

	// Вычисляем начало недели для записи в БД
	now := time.Now()
	weekStart := now.AddDate(0, 0, -7).Truncate(24 * time.Hour)

	var winners []string

	for _, a := range achs {
		description := a.Description

		log.Printf("[weekly_embed] Processing: %s (%s)", a.Code, a.Title)
		log.Printf("[weekly_embed] Default description: '%s'", description)

		// Генерируем AI-описание, если generator доступен
		// if generator != nil {
		// 	log.Printf("[weekly_embed] Calling AI.GenerateAchievementText for '%s'...", a.Code)

		// 	aiDesc, err := ai.GenerateAchievementText(generator, ai.AchievementAIContext{
		// 		Title: a.Title,
		// 		Kind:  "achievement",
		// 		Fact:  fmt.Sprintf("%s получил достижение за %s", a.Username, a.Value),
		// 	})

		// 	if err != nil {
		// 		log.Printf("[weekly_embed][AI] ❌ Error: %v", err)
		// 	} else {
		// 		description = aiDesc
		// 		log.Printf("[weekly_embed][AI] ✅ Generated: '%s'", aiDesc)
		// 	}
		// } else {
		// 	log.Println("[weekly_embed] ⚠️ Skipping AI (generator is nil)")
		// }

		// 💾 Сохраняем достижение в БД
		if store != nil {
			err := store.SaveAchievement(
				a.UserID,
				a.Username,
				a.Code,
				a.Title,
				a.Value,
				description,
				weekStart,
			)
			if err != nil {
				log.Printf("[weekly_embed][DB] ❌ Save error: %v", err)
			} else {
				log.Printf("[weekly_embed][DB] ✅ Saved: %s → %s", a.Username, a.Code)
			}
		}

		line := fmt.Sprintf(
			"**%s**\n%s\n→ <@%s> (%s)",
			a.Title,
			description,
			a.UserID,
			a.Value,
		)
		winners = append(winners, line)

		log.Printf("[weekly_embed] Final description: '%s'", description)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📊 Итоги недели",
		Description: joinOrEmpty(winners), // ← используем Description вместо Fields
		Color:       0xED4245,
	}

	log.Println("[weekly_embed] Sending embed to Discord...")
	msg, err := s.ChannelMessageSendEmbed(channelID, embed) // ← CHANGE THIS
	if err != nil {
		log.Printf("[weekly_embed] ❌ Discord API error: %v", err) // ← ADD THIS
	} else {
		log.Printf("[weekly_embed] ✅ Message sent successfully, ID: %s", msg.ID) // ← ADD THIS
	}
	log.Println("[weekly_embed] === END SendWeeklyEmbed ===")
}

func joinOrEmpty(lines []string) string {
	if len(lines) == 0 {
		return "—"
	}
	return strings.Join(lines, "\n\n")
}

// getCurrentWeekStart возвращает начало текущей недели (понедельник 00:00)
func getCurrentWeekStart() time.Time {
	now := time.Now()

	weekday := now.Weekday()

	// В Go воскресенье = 0, понедельник = 1
	// Нужно вычислить количество дней назад до понедельника
	var daysFromMonday int
	if weekday == time.Sunday {
		daysFromMonday = 6 // воскресенье → 6 дней назад до понедельника
	} else {
		daysFromMonday = int(weekday) - 1 // вторник = 2, значит 1 день назад
	}

	weekStart := now.AddDate(0, 0, -daysFromMonday)
	return time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
}
