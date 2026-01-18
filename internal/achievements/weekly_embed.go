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
	if len(achs) == 0 {
		s.ChannelMessageSend(
			channelID,
			"📊 Итоги недели\n\nНикто ничего не заслужил 😈",
		)
		return
	}

	// Вычисляем начало недели для записи в БД
	now := time.Now()
	weekStart := now.AddDate(0, 0, -7).Truncate(24 * time.Hour)

	var winners []string

	for _, a := range achs {
		description := a.Description // дефолтное описание

		// Генерируем AI-описание, если generator доступен
		if generator != nil {
			aiDesc, err := ai.GenerateAchievementText(generator, ai.AchievementAIContext{
				Title: a.Title,
				Kind:  "achievement",
				Fact:  fmt.Sprintf("%s получил достижение за %s", a.Username, a.Value),
			})
			if err != nil {
				log.Printf("[AI] generation error for %s: %v", a.Code, err)
			} else {
				description = aiDesc
			}
		}

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
				log.Printf("[DB] Failed to save achievement %s for %s: %v",
					a.Code, a.Username, err)
			} else {
				log.Printf("[DB] Saved achievement: %s → %s", a.Username, a.Code)
			}
		}

		line := fmt.Sprintf(
			"**%s**\n%s\n→ %s (%s)",
			a.Title,
			description, // ← AI-описание или дефолтное
			a.Username,
			a.Value,
		)
		winners = append(winners, line)
	}

	embed := &discordgo.MessageEmbed{
		Title: "📊 Итоги недели",
		Color: 0xED4245,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "🏆 Достижения",
				Value: joinOrEmpty(winners),
			},
		},
	}

	s.ChannelMessageSendEmbed(channelID, embed)
}

func joinOrEmpty(lines []string) string {
	if len(lines) == 0 {
		return "—"
	}
	return strings.Join(lines, "\n\n")
}
