// Package llm — минимальный клиент OpenAI-совместимого чата и офлайн-заглушка.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Message — одно сообщение чата.
type Message struct {
	Role    string `json:"role"` // system | user | assistant
	Content string `json:"content"`
}

// Responder отдаёт ответ ассистента на текущую переписку.
type Responder interface {
	Respond(ctx context.Context, msgs []Message) (string, error)
}

// Client — клиент OpenAI-совместимого шлюза (LiteLLM, vLLM, OpenAI, …).
type Client struct {
	baseURL string // корень без /v1 (или с ним — обработаем)
	apiKey  string
	model   string
	http    *http.Client
}

// New создаёт клиент. baseURL — например, https://host (клиент добавит /v1/chat/completions).
func New(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:  apiKey,
		model:   model,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Respond шлёт переписку в шлюз и возвращает текст ответа.
func (c *Client) Respond(ctx context.Context, msgs []Message) (string, error) {
	if c.baseURL == "" || c.apiKey == "" {
		return "", fmt.Errorf("llm: не заданы base_url/ключ шлюза")
	}
	body, err := json.Marshal(chatRequest{Model: c.model, Messages: msgs, Temperature: 0.4})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm: запрос: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("llm: разбор ответа: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("llm: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("llm: пустой ответ")
	}
	return out.Choices[0].Message.Content, nil
}

// endpoint склеивает URL: добавляет /v1, если его нет.
func (c *Client) endpoint() string {
	if strings.HasSuffix(c.baseURL, "/v1") {
		return c.baseURL + "/chat/completions"
	}
	return c.baseURL + "/v1/chat/completions"
}

// Offline — офлайн-заглушка для тестов и демо без сети. Не вызывает LLM,
// но соблюдает протокол: возвращает текст с меткой [[SCORE:n]].
type Offline struct{}

// Respond возвращает скриптованный ответ. Открывающее сообщение (когда ответов
// ассистента ещё нет) — вопрос без оценки; далее — разбор с оценкой 3/5.
func (Offline) Respond(_ context.Context, msgs []Message) (string, error) {
	for _, m := range msgs {
		if m.Role == "assistant" {
			return "Принял ответ. Идём глубже: какие граничные случаи и компромиссы ты видишь?\n[[SCORE:3]]", nil
		}
	}
	return "Разберём тему вместе. Начнём с вопроса: как ты сам понимаешь этот термин и где встречал его на практике?\n[[SCORE:none]]", nil
}
