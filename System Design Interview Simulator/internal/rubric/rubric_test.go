package rubric

import (
	"path/filepath"
	"testing"
)

func TestTotalAndGrade(t *testing.T) {
	card, err := Load(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	total, grade, results := card.Total([]int{3, 4, 2, 3, 3, 3})
	if total != 76.2 {
		t.Fatalf("total = %v, ожидалось 76.2", total)
	}
	if grade != "senior" {
		t.Fatalf("grade = %q, ожидалось senior", grade)
	}
	if len(results) != len(card.Dimensions) {
		t.Fatalf("results = %d, ожидалось %d", len(results), len(card.Dimensions))
	}
}

func TestGradeBoundaries(t *testing.T) {
	card, err := Load(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cases := []struct {
		total float64
		want  string
	}{
		{100, "staff"},
		{85, "staff"},
		{70, "senior"},
		{69.9, "mid"},
		{55, "mid"},
		{35, "junior"},
		{34.9, "not_ready"},
		{0, "not_ready"},
	}
	for _, c := range cases {
		if got := card.GradeFor(c.total); got != c.want {
			t.Errorf("GradeFor(%v) = %q, ожидалось %q", c.total, got, c.want)
		}
	}
}
