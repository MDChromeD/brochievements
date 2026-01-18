package achievements

import (
	"brochievements/internal/ai"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// SendWeeklyEmbed отправляет embed с еженедельными достижениями
func SendWeeklyEmbed(
	s *discordgo.Session,
	channelID string,
	achs []*WeeklyAchievement,
	generator ai.Generator, // ← добавили параметр
) {
	if len(achs) == 0 {
		s.ChannelMessageSend(
			channelID,
			"📊 Итоги недели\n\nНикто ничего не заслужил 😈",
		)
		return
	}

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
