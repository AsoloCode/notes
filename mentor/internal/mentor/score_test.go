package mentor

import "testing"

func TestParseScore(t *testing.T) {
	cases := []struct {
		in        string
		wantClean string
		wantScore int
		wantOK    bool
	}{
		{"Отличный ответ про GC.\n[[SCORE:5]]", "Отличный ответ про GC.", 5, true},
		{"Частично верно [[SCORE:3]]", "Частично верно", 3, true},
		{"[[SCORE:none]]", "", 0, false},
		{"Нет метки вообще", "Нет метки вообще", 0, false},
		{"Хвост [[SCORE:9]]", "Хвост", 5, true}, // clamp
	}
	for _, c := range cases {
		clean, score, ok := parseScore(c.in)
		if clean != c.wantClean || score != c.wantScore || ok != c.wantOK {
			t.Errorf("parseScore(%q) = (%q,%d,%v), ожидалось (%q,%d,%v)",
				c.in, clean, score, ok, c.wantClean, c.wantScore, c.wantOK)
		}
	}
}

func TestModeByID(t *testing.T) {
	if _, ok := ModeByID("INTERVIEW"); !ok {
		t.Error("ModeByID регистронезависимо не нашёл interview")
	}
	if _, ok := ModeByID("nope"); ok {
		t.Error("ModeByID нашёл несуществующий режим")
	}
	if len(Modes) != 7 {
		t.Errorf("режимов %d, ожидалось 7", len(Modes))
	}
}
