// Package config собирает настройки ментора из флагов и окружения.
// Секреты в код не зашиваются: ключ берётся из окружения.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config — эффективные настройки прогона.
type Config struct {
	Root     string // корень, где лежат progress/ и sessions/ (обычно каталог mentor/)
	BaseURL  string // OpenAI-совместимый шлюз (без /v1 — клиент сам добавит)
	APIKey   string // ключ шлюза
	Model    string // модель чата
	Color    bool   // цветной вывод
	Language string // язык объяснений (по умолчанию ru)
}

// DefaultModel — модель по умолчанию, если ничего не задано.
const DefaultModel = "gpt-4o-mini"

// Load собирает конфиг. Приоритет: флаги (уже в cfg) > env > дефолты.
// Флаги проставляет вызывающая сторона до вызова Load (пустые поля заполняются).
func Load(flagRoot, flagBaseURL, flagKey, flagModel string) (*Config, error) {
	c := &Config{
		Root:     firstNonEmpty(flagRoot, os.Getenv("MENTOR_ROOT"), "."),
		BaseURL:  firstNonEmpty(flagBaseURL, os.Getenv("MENTOR_BASE_URL"), os.Getenv("VIDEOSC_OPENAI_BASE_URL")),
		APIKey:   firstNonEmpty(flagKey, os.Getenv("MENTOR_API_KEY"), os.Getenv("OPENAI_API_KEY"), os.Getenv("LITELLM_API_KEY")),
		Model:    firstNonEmpty(flagModel, os.Getenv("MENTOR_MODEL"), DefaultModel),
		Language: firstNonEmpty(os.Getenv("MENTOR_LANG"), "ru"),
	}
	abs, err := filepath.Abs(c.Root)
	if err != nil {
		return nil, fmt.Errorf("config: root: %w", err)
	}
	c.Root = abs
	c.Color = os.Getenv("NO_COLOR") == "" && isTerminal()
	return c, nil
}

// ProgressDir — каталог файлов прогресса.
func (c *Config) ProgressDir() string { return filepath.Join(c.Root, "progress") }

// SessionsDir — каталог журнала сессий.
func (c *Config) SessionsDir() string { return filepath.Join(c.Root, "sessions") }

// LLMReady — готов ли клиент к сетевому вызову (есть base_url и ключ).
func (c *Config) LLMReady() bool {
	return strings.TrimSpace(c.BaseURL) != "" && strings.TrimSpace(c.APIKey) != ""
}

// LLMHint — что дописать в окружение, если LLM не настроен.
func (c *Config) LLMHint() string {
	if c.BaseURL == "" {
		return "не задан шлюз: export MENTOR_BASE_URL=https://<эндпоинт-без-/v1>"
	}
	if c.APIKey == "" {
		return "не задан ключ: export MENTOR_API_KEY=... (или OPENAI_API_KEY/LITELLM_API_KEY)"
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func isTerminal() bool {
	st, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}
