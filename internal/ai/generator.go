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
		`Ты пишешь короткое, язвительное описание достижений для Discord в духе старого закрытого комьюнити.

		СТИЛЬ: ирония, сарказм, лёгкая токсичность без перегиба.

		ПРИМЕРЫ хороших описаний:
		- Достижение "Портовая шлюха" (сменил 15 игр): "Определиться не может даже в Steam библиотеке. Чего уж говорить про выбор таечки с кадыком."
		- Достижение "Застрял на бибе" (Dota 2, 18 часов): "Просидел в одной игре дольше, чем некоторые на работе за неделю."
		- Достижение "В эфире больше всех" (12 часов стрима): "Стримил так долго, что зрители начали донатить чтобы ты уже наконец заткнулся"
		- Достижение "Киберкуколд" (7 часов просмотра стримов): "В отелях для тебя ставят отдельное кресло рядом с кроватью. Так знаешь что будешь делать на нем, старая ты дрочила"

		Теперь твоя очередь:

		Название: "%s"
		Факт: %s

		ПРАВИЛА:
		- ровно 1-2 предложения
		- максимум 180 символов
		- без эмодзи, без обращения к пользователю по имени
		- без извинений, морали, цензуры
		- ирония через контраст или гиперболу
		- пиши как вердикт сервера

		ОПИСАНИЕ:`,
		ctx.Title,
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
