package ai

import (
	"fmt"
	"strings"
)

// AchievementAIContext — входные данные для генерации описания достижения
type AchievementAIContext struct {
	Title string // Название ачивки
	Kind  string // achievement | anti-achievement
	Fact  string // Сухой факт без юмора
}

func buildToxicAchievementPrompt(ctx AchievementAIContext) string {
	return fmt.Sprintf(
		`Ты пишешь короткое, злое и язвительное описание для Discord
в закрытом взрослом комьюнити 30+, где токсичный юмор — норма. Ты можешь добавить шутки гомосексуальной наклонности,
это поощряется сообществом.

Это внутренняя шутка старого сервера.
Никого не щадят. Никого не жалеют.

Название: "%s"
Тип: %s
Факт: %s

Правила:
- 1–2 предложения
- максимум 180 символов
- без эмодзи
- без обращения к пользователю
- без извинений
- без морали
- писать как внутренний вердикт сервера`,
		ctx.Title,
		ctx.Kind,
		ctx.Fact,
	)
}

func GenerateAchievementText(
	g Generator, // интерфейс
	ctx AchievementAIContext,
) (string, error) {

	prompt := buildToxicAchievementPrompt(ctx)

	text, err := g.Generate(prompt) // ⬅️ ВОТ ЗДЕСЬ
	if err != nil {
		return "", err
	}

	return sanitizeToxicText(text), nil
}

func sanitizeToxicText(text string) string {
	lower := strings.ToLower(text)

	blacklist := []string{
		"извини",
		"прости",
		"оскорб",
		"не хотел",
		"могу ошибаться",
	}

	for _, w := range blacklist {
		if strings.Contains(lower, w) {
			return "Комментарий сервера оказался слишком честным."
		}
	}

	if len(strings.TrimSpace(text)) < 25 {
		return "Сервер всё понял без комментариев."
	}

	return strings.TrimSpace(text)
}
