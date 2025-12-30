package achievements

import (
	"fmt"
)

type Achievement struct {
	Title       string
	Description string
	Username    string
	Value       string
	Period      string
}

func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60

	return fmt.Sprintf("%dh %dm", h, m)
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
