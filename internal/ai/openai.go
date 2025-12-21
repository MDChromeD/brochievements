package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
)

type OpenAIGenerator struct {
	apiKey string
}

func NewOpenAIGenerator() (*OpenAIGenerator, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, errors.New("OPENAI_API_KEY not set")
	}

	return &OpenAIGenerator{apiKey: key}, nil
}

func (g *OpenAIGenerator) Generate(prompt string) (string, error) {
	payload := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "Ты пишешь короткие, смешные и дружелюбные достижения для Discord-сервера небольшого комьюнити.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.8,
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(
		"POST",
		"https://api.openai.com/v1/chat/completions",
		bytes.NewBuffer(body),
	)

	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", errors.New("empty OpenAI response")
	}

	return result.Choices[0].Message.Content, nil
}
