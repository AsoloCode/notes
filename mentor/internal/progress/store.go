// Package progress хранит мастерство по темам и ведёт расписание повторений.
// Файлы: progress.json (машиночитаемо) и progress.md (для человека/Obsidian).
package progress

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/AsoloCode/notes/mentor/internal/curriculum"
)

// Review — одна оценённая попытка.
type Review struct {
	At      time.Time `json:"at"`
	Mode    string    `json:"mode"`
	Score   int       `json:"score"` // 0..5
	Correct bool      `json:"correct"`
	Note    string    `json:"note,omitempty"`
}

// Record — прогресс по одной теме.
type Record struct {
	Direction string    `json:"direction"`
	Topic     string    `json:"topic"`
	Mastery   int       `json:"mastery"` // 0..5
	Attempts  int       `json:"attempts"`
	Correct   int       `json:"correct"`
	Streak    int       `json:"streak"`
	ActiveDay int       `json:"active_days"` // сколько разных дней занимались темой
	LastDay   string    `json:"last_day"`
	LastSeen  time.Time `json:"last_seen"`
	NextDue   time.Time `json:"next_due"`
	Reviews   []Review  `json:"reviews"`
}

// Store — весь прогресс.
type Store struct {
	Version int                `json:"version"`
	Updated time.Time          `json:"updated"`
	Records map[string]*Record `json:"records"`
}

// masteredFrom — порог, с которого тема считается «изученной».
const masteredFrom = 4

const maxReviews = 30

// New создаёт пустое хранилище.
func New() *Store {
	return &Store{Version: 1, Records: map[string]*Record{}}
}

// Load читает progress.json; отсутствие файла — не ошибка (пустой прогресс).
func Load(path string) (*Store, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return New(), nil
		}
		return nil, fmt.Errorf("progress: чтение %s: %w", path, err)
	}
	s := New()
	if err := json.Unmarshal(raw, s); err != nil {
		return nil, fmt.Errorf("progress: разбор %s: %w", path, err)
	}
	if s.Records == nil {
		s.Records = map[string]*Record{}
	}
	return s, nil
}

// SaveJSON пишет progress.json атомарно.
func (s *Store) SaveJSON(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Get возвращает запись темы (или nil).
func (s *Store) Get(direction, topic string) *Record {
	return s.Records[curriculum.Key(direction, topic)]
}

// Update применяет результат одной попытки к записи темы и пересчитывает мастерство.
func (s *Store) Update(direction, topic, mode string, score int, note string, now time.Time) *Record {
	key := curriculum.Key(direction, topic)
	r := s.Records[key]
	if r == nil {
		r = &Record{Direction: direction, Topic: topic}
		s.Records[key] = r
	}
	if score < 0 {
		score = 0
	}
	if score > 5 {
		score = 5
	}
	correct := score >= 4

	r.Attempts++
	if correct {
		r.Correct++
		r.Streak++
	} else {
		r.Streak = 0
	}
	day := now.Format("2006-01-02")
	if r.LastDay != day {
		r.ActiveDay++
		r.LastDay = day
	}
	r.LastSeen = now
	r.Mastery = masteryFor(r.Streak, r.ActiveDay)
	r.NextDue = now.Add(intervalFor(r.Mastery))

	r.Reviews = append(r.Reviews, Review{At: now, Mode: mode, Score: score, Correct: correct, Note: note})
	if len(r.Reviews) > maxReviews {
		r.Reviews = r.Reviews[len(r.Reviews)-maxReviews:]
	}
	s.Updated = now
	return r
}

// masteryFor пересчитывает уровень мастерства. Важно: одного верного ответа
// недостаточно — для уровня «изучено» (>=4) нужен streak и работа в разные дни.
func masteryFor(streak, activeDays int) int {
	switch {
	case streak >= 5 && activeDays >= 3:
		return 5
	case streak >= 4 && activeDays >= 2:
		return 4
	case streak >= 3 && activeDays >= 2:
		return 3
	case streak >= 2:
		return 2
	case streak >= 1:
		return 1
	default:
		return 0
	}
}

// intervalFor — пауза до следующего повторения по уровню мастерства.
func intervalFor(mastery int) time.Duration {
	steps := []time.Duration{
		6 * time.Hour,
		24 * time.Hour,
		48 * time.Hour,
		96 * time.Hour,
		7 * 24 * time.Hour,
		14 * 24 * time.Hour,
	}
	if mastery < 0 {
		mastery = 0
	}
	if mastery >= len(steps) {
		mastery = len(steps) - 1
	}
	return steps[mastery]
}

// Key — ключ темы "<direction>/<topic>".
type Key struct {
	Direction string
	Topic     string
	Record    *Record
}

// DueTopic — тема, которую пора повторить.
type DueTopic struct {
	Direction string
	Topic     string
	Record    *Record
}

// Due возвращает темы, срок повторения которых наступил (или ещё не занимались),
// отсортированные по срочности.
func (s *Store) Due(now time.Time) []DueTopic {
	var out []DueTopic
	for _, r := range s.Records {
		if !r.NextDue.After(now) {
			out = append(out, DueTopic{r.Direction, r.Topic, r})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Record.NextDue.Equal(out[j].Record.NextDue) {
			return out[i].Record.Mastery < out[j].Record.Mastery
		}
		return out[i].Record.NextDue.Before(out[j].Record.NextDue)
	})
	return out
}

// Weak возвращает N самых слабых тем (по мастерству, затем по давности).
func (s *Store) Weak(n int) []DueTopic {
	var out []DueTopic
	for _, r := range s.Records {
		out = append(out, DueTopic{r.Direction, r.Topic, r})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Record.Mastery == out[j].Record.Mastery {
			return out[i].Record.LastSeen.Before(out[j].Record.LastSeen)
		}
		return out[i].Record.Mastery < out[j].Record.Mastery
	})
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// MasteredTopics — сколько тем достигло порога «изучено».
func (s *Store) MasteredTopics() int {
	n := 0
	for _, r := range s.Records {
		if r.Mastery >= masteredFrom {
			n++
		}
	}
	return n
}

// RenderMD рендерит progress.md: сводка + таблица по направлениям.
func (s *Store) RenderMD(cur *curriculum.Curriculum) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("type: progress\n")
	b.WriteString("tool: mentor\n")
	b.WriteString(fmt.Sprintf("updated: %s\n", time.Now().Format("2006-01-02")))
	b.WriteString("---\n\n")
	b.WriteString("# Прогресс обучения\n\n")
	b.WriteString("> Автогенерируется CLI-ментором. Правки вручную затрутся при следующем прогоне.\n\n")

	total, mastered := 0, 0
	if cur != nil {
		for _, d := range cur.Directions {
			total += len(d.Topics)
		}
	}
	mastered = s.MasteredTopics()
	b.WriteString(fmt.Sprintf("Тем всего: **%d**, изучено (мастерство ≥ 4): **%d**, затронуто: **%d**.\n\n", total, mastered, len(s.Records)))

	bar := func(m int) string {
		filled := strings.Repeat("●", m) + strings.Repeat("○", 5-m)
		return filled
	}

	if cur != nil {
		for _, d := range cur.Directions {
			b.WriteString(fmt.Sprintf("## %s\n\n", d.Title))
			b.WriteString("| Тема | Мастерство | Попытки | Верных | Следующее |\n")
			b.WriteString("|---|---|---|---|---|\n")
			any := false
			for _, t := range d.Topics {
				r := s.Get(d.ID, t.ID)
				if r == nil {
					continue
				}
				any = true
				b.WriteString(fmt.Sprintf("| %s | %s (%d) | %d | %d | %s |\n",
					t.Title, bar(r.Mastery), r.Mastery, r.Attempts, r.Correct, dueStr(r.NextDue)))
			}
			if !any {
				b.WriteString("| _ещё не начато_ | | | | |\n")
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

func dueStr(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2006-01-02 15:04")
}
