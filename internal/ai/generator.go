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
		`Ты пишешь короткие, ядовито-ироничные описания достижений для Discord-сервера в стиле "токсичного гороскопа".

СТРУКТУРА:
1. Начало: звучит как обычное наблюдение или предсказание
2. Развитие: добавляет абсурдную конкретику
3. Панчлайн: резкий поворот в неожиданную/унизительную сторону

СТИЛЬ:
- Токсичная доброжелательность ("помни, ты не такое уж дерьмо")
- Обман ожиданий (начинается позитивно → становится хуже)
- Абсурдная конкретика ("прохожие будут подкармливать объедками")
- Ложная мотивация ("не может столько людей ошибаться")

ПРИМЕРЫ ХОРОШИХ ОПИСАНИЙ:
- "Решаешь пройти по новому маршруту, но заблудишься и потеряешься. Будешь блуждать по улицам, как бродячая собака. Прохожие будут тебя подкармливать объедками."
- "Тебе будет казаться, что сегодня ты классно выглядишь. Все остальные будут думать, что ты урод. Как и в любой другой день."
- "Узнаешь, что есть точно такой же чат, как этот, но без тебя. Там все постоянно общаются, встречаются, веселятся."

ПРАВИЛА:
- 1-3 предложения, максимум 200 символов
- Без прямых оскорблений, только ирония
- Без извинений и морали
- Без эмодзи
- Конкретика важнее абстракций

Теперь твоя очередь:

Достижение: "%s"
Факт: %s

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
