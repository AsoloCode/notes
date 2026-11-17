// Package curriculum хранит карту направлений и тем, по которым учит ментор.
// Данные лежат рядом в curriculum.json (схема стабильна, правится без кода).
package curriculum

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed curriculum.json
var raw []byte

// Topic — одна тема внутри направления.
type Topic struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Material  string   `json:"material,omitempty"`   // путь к конспекту в репозитории (опц.)
	KeyPoints []string `json:"key_points,omitempty"` // опорные тезисы для вопросов
}

// Direction — направление (Go, PostgreSQL, …).
type Direction struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Goal   string  `json:"goal"`
	Topics []Topic `json:"topics"`
}

// Curriculum — весь набор направлений.
type Curriculum struct {
	Directions []Direction `json:"directions"`
}

// Load читает встроенный curriculum.json.
func Load() (*Curriculum, error) {
	var c Curriculum
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("curriculum: разбор встроенных данных: %w", err)
	}
	if len(c.Directions) == 0 {
		return nil, fmt.Errorf("curriculum: пустой список направлений")
	}
	return &c, nil
}

// Direction ищет направление по id или title (регистронезависимо).
func (c *Curriculum) Direction(id string) (Direction, bool) {
	key := strings.ToLower(strings.TrimSpace(id))
	for _, d := range c.Directions {
		if strings.ToLower(d.ID) == key || strings.ToLower(d.Title) == key {
			return d, true
		}
	}
	return Direction{}, false
}

// Topic ищет тему внутри направления по id или title.
func (d Direction) Topic(id string) (Topic, bool) {
	key := strings.ToLower(strings.TrimSpace(id))
	for _, t := range d.Topics {
		if strings.ToLower(t.ID) == key || strings.ToLower(t.Title) == key {
			return t, true
		}
	}
	return Topic{}, false
}

// Key — стабильный ключ темы для файла прогресса: "<direction>/<topic>".
func Key(directionID, topicID string) string {
	return directionID + "/" + topicID
}
