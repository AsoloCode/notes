package progress

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/AsoloCode/notes/mentor/internal/curriculum"
)

func TestMasteryRules(t *testing.T) {
	s := New()
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	// Один верный ответ НЕ делает тему изученной.
	r := s.Update("go", "slices", "recall", 5, "", base)
	if r.Mastery >= 4 {
		t.Fatalf("после одного верного ответа mastery=%d, ожидалось < 4", r.Mastery)
	}
	if r.Streak != 1 || r.ActiveDay != 1 {
		t.Fatalf("streak/days = %d/%d, ожидалось 1/1", r.Streak, r.ActiveDay)
	}

	// Второй верный в тот же день — максимум 2 (нужны разные дни для 3+).
	r = s.Update("go", "slices", "recall", 5, "", base.Add(time.Hour))
	if r.Mastery != 2 {
		t.Fatalf("в один день mastery=%d, ожидалось 2", r.Mastery)
	}

	// Закрепление в разные дни поднимает уровень.
	day2 := base.Add(24 * time.Hour)
	r = s.Update("go", "slices", "recall", 5, "", day2)
	r = s.Update("go", "slices", "recall", 5, "", day2.Add(time.Hour))
	if r.Mastery != 4 {
		t.Fatalf("streak=4 days=2 → mastery=%d, ожидалось 4", r.Mastery)
	}
	day3 := base.Add(48 * time.Hour)
	r = s.Update("go", "slices", "recall", 5, "", day3)
	if r.Mastery != 5 {
		t.Fatalf("streak=5 days=3 → mastery=%d, ожидалось 5", r.Mastery)
	}

	// Ошибка сбрасывает серию.
	r = s.Update("go", "slices", "recall", 1, "", day3.Add(time.Hour))
	if r.Streak != 0 || r.Mastery != 0 {
		t.Fatalf("после ошибки streak=%d mastery=%d, ожидалось 0/0", r.Streak, r.Mastery)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	s := New()
	now := time.Now().Round(0)
	s.Update("postgresql", "mvcc", "learn", 4, "", now)
	path := filepath.Join(t.TempDir(), "progress.json")
	if err := s.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	r := got.Get("postgresql", "mvcc")
	if r == nil || r.Attempts != 1 || r.Mastery == 0 {
		t.Fatalf("после Load запись потеряна/пуста: %+v", r)
	}
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Records) != 0 {
		t.Fatalf("ожидался пустой прогресс, got %d", len(s.Records))
	}
}

func TestDueAndWeak(t *testing.T) {
	s := New()
	past := time.Now().Add(-time.Hour)
	s.Update("go", "maps", "recall", 5, "", past) // NextDue в будущем
	future := time.Now().Add(72 * time.Hour)
	s.Update("go", "gc", "recall", 0, "", future)
	// искусственно сдвинем NextDue "maps" в прошлое
	rec := s.Get("go", "maps")
	rec.NextDue = time.Now().Add(-time.Minute)

	due := s.Due(time.Now())
	if len(due) != 1 || due[0].Topic != "maps" {
		t.Fatalf("due = %+v, ожидалась только maps", due)
	}
	weak := s.Weak(5)
	if len(weak) == 0 || weak[0].Topic != "gc" {
		t.Fatalf("weak[0] = %+v, ожидалась gc (mastery 0)", weak)
	}
}

func TestRenderMD(t *testing.T) {
	cur, err := curriculum.Load()
	if err != nil {
		t.Fatal(err)
	}
	s := New()
	s.Update("go", "slices", "recall", 5, "", time.Now())
	md := s.RenderMD(cur)
	if len(md) == 0 {
		t.Fatal("пустой RenderMD")
	}
	for _, want := range []string{"# Прогресс обучения", "## Go", "Массивы и срезы"} {
		if !contains(md, want) {
			t.Fatalf("в progress.md нет %q", want)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
