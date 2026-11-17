package curriculum

import "testing"

func TestLoadAndLookup(t *testing.T) {
	cur, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cur.Directions) != 5 {
		t.Fatalf("направлений %d, ожидалось 5", len(cur.Directions))
	}
	d, ok := cur.Direction("go")
	if !ok {
		t.Fatal("направление go не найдено")
	}
	tp, ok := d.Topic("slices")
	if !ok || tp.Title != "Массивы и срезы" {
		t.Fatalf("тема slices не найдена: %+v", tp)
	}
	if got := Key("go", "slices"); got != "go/slices" {
		t.Fatalf("Key = %q", got)
	}
	// поиск по названию, регистронезависимо
	if _, ok := d.Topic("МАССИВЫ И СРЕЗЫ"); !ok {
		t.Fatal("поиск темы по названию не работает")
	}
}

func TestEveryDirectionHasTopics(t *testing.T) {
	cur, _ := Load()
	for _, d := range cur.Directions {
		if len(d.Topics) == 0 {
			t.Errorf("направление %s без тем", d.ID)
		}
		for _, tp := range d.Topics {
			if tp.ID == "" || tp.Title == "" {
				t.Errorf("тема без id/title в %s: %+v", d.ID, tp)
			}
		}
	}
}
