// Package rubric загружает оценочный лист (rubric/scorecard.json) и считает балл.
package rubric

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
)

// Scorecard — корневой объект rubric/scorecard.json.
type Scorecard struct {
	Version    string      `json:"version"`
	Scale      Scale       `json:"scale"`
	Dimensions []Dimension `json:"dimensions"`
	Grades     []Grade     `json:"grades"`
}

// Scale — диапазон шкалы оценок.
type Scale struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// Dimension — одно измерение рубрики.
type Dimension struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Weight int    `json:"weight"`
}

// Grade — порог грейда.
type Grade struct {
	Grade    string  `json:"grade"`
	MinTotal float64 `json:"min_total"`
}

// Load читает оценочный лист из <root>/rubric/scorecard.json.
func Load(root string) (*Scorecard, error) {
	path := filepath.Join(root, "rubric", "scorecard.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать оценочный лист %s: %w", path, err)
	}
	var s Scorecard
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("не удалось разобрать оценочный лист %s: %w", path, err)
	}
	if len(s.Dimensions) == 0 {
		return nil, fmt.Errorf("в оценочном листе нет измерений: %s", path)
	}
	return &s, nil
}

// Result — вклад одного измерения и итог.
type Result struct {
	Dimension    Dimension
	Score        int
	Contribution float64
}

// Total — суммарный балл и грейд по оценкам (в порядке Dimensions).
func (s *Scorecard) Total(scores []int) (float64, string, []Result) {
	max := float64(s.Scale.Max)
	results := make([]Result, 0, len(s.Dimensions))
	var total float64
	for i, d := range s.Dimensions {
		contribution := float64(scores[i]) / max * float64(d.Weight)
		total += contribution
		results = append(results, Result{Dimension: d, Score: scores[i], Contribution: contribution})
	}
	total = round1(total)
	return total, s.GradeFor(total), results
}

// GradeFor возвращает грейд для итогового балла (первый подходящий сверху вниз).
func (s *Scorecard) GradeFor(total float64) string {
	grades := make([]Grade, len(s.Grades))
	copy(grades, s.Grades)
	sort.SliceStable(grades, func(i, j int) bool { return grades[i].MinTotal > grades[j].MinTotal })
	for _, g := range grades {
		if total >= g.MinTotal {
			return g.Grade
		}
	}
	return "not_ready"
}

// round1 округляет до одного знака по правилу «половина к чётному» (банковское).
func round1(v float64) float64 {
	scaled := v * 10
	floor := math.Floor(scaled)
	diff := scaled - floor
	switch {
	case diff < 0.5:
		return floor / 10
	case diff > 0.5:
		return (floor + 1) / 10
	default:
		if math.Mod(floor, 2) == 0 {
			return floor / 10
		}
		return (floor + 1) / 10
	}
}
