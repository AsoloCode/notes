// Package bank загружает машиночитаемый банк сценариев (data/scenarios.json).
package bank

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Bank — корневой объект data/scenarios.json.
type Bank struct {
	Schema    string     `json:"schema"`
	Rubric    string     `json:"rubric"`
	Scenarios []Scenario `json:"scenarios"`
}

// Scenario — один сценарий симулятора.
type Scenario struct {
	Slug                 string            `json:"slug"`
	Order                int               `json:"order"`
	Title                string            `json:"title"`
	Level                string            `json:"level"`
	Focus                string            `json:"focus"`
	ScenarioPath         string            `json:"scenario_path"`
	SolutionPath         string            `json:"solution_path"`
	Functional           []string          `json:"functional"`
	NonFunctional        []string          `json:"non_functional"`
	InterviewerQuestions []string          `json:"interviewer_questions"`
	KeyMetrics           map[string]string `json:"key_metrics"`
	Components           []string          `json:"components"`
	Consistency          map[string]string `json:"consistency"`
	Bottlenecks          []string          `json:"bottlenecks"`
	TradeOffs            []string          `json:"trade_offs"`
	RedFlags             []string          `json:"red_flags"`
	StrongSignals        []string          `json:"strong_signals"`
}

// Load читает банк из <root>/data/scenarios.json.
func Load(root string) (*Bank, error) {
	path := filepath.Join(root, "data", "scenarios.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать банк %s: %w", path, err)
	}
	var b Bank
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, fmt.Errorf("не удалось разобрать банк %s: %w", path, err)
	}
	if len(b.Scenarios) == 0 {
		return nil, fmt.Errorf("в банке нет сценариев: %s", path)
	}
	return &b, nil
}

// Find возвращает сценарий по slug.
func (b *Bank) Find(slug string) (*Scenario, bool) {
	for i := range b.Scenarios {
		if b.Scenarios[i].Slug == slug {
			return &b.Scenarios[i], true
		}
	}
	return nil, false
}

// AttemptFiles возвращает отсортированный список файлов попыток для slug.
// Файл .gitkeep игнорируется.
func AttemptFiles(root, slug string) []string {
	pattern := filepath.Join(root, "attempts", "*"+slug+"*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if filepath.Base(m) == ".gitkeep" {
			continue
		}
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}
