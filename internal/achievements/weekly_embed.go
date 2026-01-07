package achievements

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func sendWeeklyEmbed(
	s *discordgo.Session,
	channelID string,
	achs []*WeeklyAchievement,
) {
	if len(achs) == 0 {
		s.ChannelMessageSend(
			channelID,
			"📊 Итоги недели\n\nНикто ничего не заслужил 😈",
		)
		return
	}

	var winners []string
	// var shamed []string

	for _, a := range achs {
		line := fmt.Sprintf(
			"**%s**\n→ %s (%s)",
			a.Title,
			a.Username,
			a.Value,
		)

		// грубое, но рабочее разделение
		// if a.Code == "ghost" ||
		// 	a.Code == "afk_life" ||
		// 	a.Code == "just_looking" ||
		// 	a.Code == "launched_and_left" {
		// 	shamed = append(shamed, line)
		// } else {
		winners = append(winners, line)
		// }
	}

	embed := &discordgo.MessageEmbed{
		Title: "📊 Итоги недели",
		Color: 0xED4245,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "🏆 Достижения",
				Value: joinOrEmpty(winners),
			},
			// {
			// 	Name:  "💀 Антиачивки",
			// 	Value: joinOrEmpty(shamed),
			// },
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
