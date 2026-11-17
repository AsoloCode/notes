// Package validate реализует структурную валидацию банка сценариев и оценочного листа.
package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AsoloCode/notes/sdsim/internal/bank"
	"github.com/AsoloCode/notes/sdsim/internal/rubric"
)

// ValidateBank проверяет целостность <root>/data/scenarios.json и rubric/scorecard.json.
// Возвращает список проблем (пустой — если всё валидно).
func ValidateBank(root string) []string {
	var problems []string

	b, err := bank.Load(root)
	if err != nil {
		return []string{err.Error()}
	}

	orders := make([]int, 0, len(b.Scenarios))
	slugs := make(map[string]bool)

	for i := range b.Scenarios {
		sc := &b.Scenarios[i]
		where := fmt.Sprintf("scenarios[%d] (%s)", i, sc.Slug)

		if sc.Slug == "" {
			problems = append(problems, fmt.Sprintf("scenarios[%d]: пустой slug", i))
		}
		if sc.Title == "" {
			problems = append(problems, fmt.Sprintf("%s: пустой title", where))
		}
		orders = append(orders, sc.Order)
		if slugs[sc.Slug] {
			problems = append(problems, fmt.Sprintf("%s: дубликат slug", where))
		}
		slugs[sc.Slug] = true

		checkNonEmptyList(&problems, where, "functional", sc.Functional)
		checkNonEmptyList(&problems, where, "non_functional", sc.NonFunctional)
		checkNonEmptyList(&problems, where, "interviewer_questions", sc.InterviewerQuestions)
		checkNonEmptyList(&problems, where, "components", sc.Components)
		checkNonEmptyList(&problems, where, "bottlenecks", sc.Bottlenecks)
		checkNonEmptyList(&problems, where, "trade_offs", sc.TradeOffs)
		checkNonEmptyList(&problems, where, "red_flags", sc.RedFlags)
		checkNonEmptyList(&problems, where, "strong_signals", sc.StrongSignals)

		if len(sc.Consistency) == 0 {
			problems = append(problems, fmt.Sprintf("%s: 'consistency' должен быть непустым", where))
		} else {
			for k, v := range sc.Consistency {
				if strings.TrimSpace(v) == "" {
					problems = append(problems, fmt.Sprintf("%s: consistency[%s] пустое значение", where, k))
				}
			}
		}

		checkPath(&problems, root, where, "scenario_path", sc.ScenarioPath, sc.Slug)
		checkPath(&problems, root, where, "solution_path", sc.SolutionPath, sc.Slug)
	}

	// order должен быть 1..N по возрастанию
	seqOK := len(orders) > 0
	for i, o := range orders {
		if o != i+1 {
			seqOK = false
			break
		}
	}
	if !seqOK {
		problems = append(problems, fmt.Sprintf("order должен быть 1..N по порядку, получено: %v", orders))
	}

	// scorecard
	card, err := rubric.Load(root)
	if err != nil {
		problems = append(problems, err.Error())
	} else {
		if len(card.Dimensions) != 6 {
			problems = append(problems, fmt.Sprintf("scorecard: ожидалось 6 измерений, найдено %d", len(card.Dimensions)))
		}
		sum := 0
		for _, d := range card.Dimensions {
			sum += d.Weight
		}
		if sum != 100 {
			problems = append(problems, fmt.Sprintf("scorecard: сумма весов = %d, ожидалось 100", sum))
		}
	}

	return problems
}

func checkNonEmptyList(problems *[]string, where, key string, list []string) {
	if len(list) == 0 {
		*problems = append(*problems, fmt.Sprintf("%s: '%s' должен быть непустым списком", where, key))
	}
}

func checkPath(problems *[]string, root, where, key, rel, slug string) {
	if rel == "" {
		*problems = append(*problems, fmt.Sprintf("%s: пустой '%s'", where, key))
		return
	}
	if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
		*problems = append(*problems, fmt.Sprintf("%s: файл '%s' (%s) не найден", where, rel, key))
		return
	}
	// slug должен совпадать с суффиксом имени файла: NN-<slug>.md
	stem := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
	if idx := strings.Index(stem, "-"); idx >= 0 {
		stem = stem[idx+1:]
	}
	if stem != slug {
		*problems = append(*problems, fmt.Sprintf("%s: slug не совпадает с именем файла '%s'", where, rel))
	}
}
