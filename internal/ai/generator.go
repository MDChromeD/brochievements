package ai

// Generator — интерфейс генерации текста достижений.
// Реализации (OpenAI, локальная модель и т.д.) должны
// возвращать готовый текст по prompt.
type Generator interface {
	Generate(prompt string) (string, error)
}
