package achievements

import (
	"fmt"

	"brochievements/internal/storage"
)

type Achievement struct {
	Title       string
	Description string
	Username    string
	Value       string
	Period      string
}

func VoiceMaster(stat *storage.VoiceTimeStat) Achievement {
	return Achievement{
		Title:    "🎧 Хозяин голосового канала",
		Username: stat.Username,
		Value:    formatDuration(stat.Seconds),
		Period:   "неделя",
		Description: fmt.Sprintf(
			"%s провёл в голосовых каналах больше всех — %s за неделю.",
			stat.Username,
			formatDuration(stat.Seconds),
		),
	}
}

func FrequentVisitor(stat *storage.VoiceJoinStat) Achievement {
	return Achievement{
		Title:    "🚪 Частый гость",
		Username: stat.Username,
		Value:    fmt.Sprintf("%d входов", stat.Count),
		Period:   "неделя",
		Description: fmt.Sprintf(
			"%s заходил в голосовые каналы чаще всех — %d раз за неделю.",
			stat.Username,
			stat.Count,
		),
	}
}

func Marathoner(stat *storage.LongestVoiceSessionStat) Achievement {
	return Achievement{
		Title:    "⏱ Марафонец",
		Username: stat.Username,
		Value:    formatDuration(stat.Seconds),
		Period:   "неделя",
		Description: fmt.Sprintf(
			"%s провёл в одном голосовом канале рекордное время — %s.",
			stat.Username,
			formatDuration(stat.Seconds),
		),
	}
}

func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60

	return fmt.Sprintf("%dh %dm", h, m)
}

func GameFan(stat *storage.GameStat) Achievement {
	return Achievement{
		Title:    "🎮 Преданный фанат",
		Period:   "неделя",
		Username: stat.Username,
		Value:    stat.Game,
		Description: fmt.Sprintf(
			"%s чаще всех был замечен в игре **%s**.",
			stat.Username,
			stat.Game,
		),
	}
}

func (a Achievement) Prompt() string {
	return `
Название достижения: ` + a.Title + `
Победитель: ` + a.Username + `
Значение: ` + a.Value + `
Период: ` + a.Period + `

Сформулируй короткое, смешное и дружелюбное описание для Discord.
`
}
