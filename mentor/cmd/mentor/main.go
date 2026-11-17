// Command mentor — персональный CLI-ментор для подготовки к собеседованиям.
//
// Направления: Go, PostgreSQL, алгоритмы, System Design, микросервисы.
// Режимы: обучение темы, active recall, mock interview, разбор ошибки,
// повторение слабых тем, проверка решения, production case.
//
// Примеры:
//
//	mentor                      # интерактивное меню
//	mentor --list               # направления и темы
//	mentor --due                # что пора повторить
//	mentor --progress           # показать прогресс
//	mentor --direction go --topic slices --mode recall
//	mentor --demo               # офлайн-прогон без сети (для проверки)
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/AsoloCode/notes/mentor/internal/config"
	"github.com/AsoloCode/notes/mentor/internal/curriculum"
	"github.com/AsoloCode/notes/mentor/internal/llm"
	"github.com/AsoloCode/notes/mentor/internal/mentor"
	"github.com/AsoloCode/notes/mentor/internal/progress"
	"github.com/AsoloCode/notes/mentor/internal/ui"
)

const version = "1.0.0"

func main() {
	fs := flag.NewFlagSet("mentor", flag.ExitOnError)
	root := fs.String("root", "", "корень с progress/ и sessions/ (по умолчанию текущий каталог)")
	baseURL := fs.String("base-url", "", "URL OpenAI-совместимого шлюза (без /v1)")
	model := fs.String("model", "", "модель чата")
	apiKey := fs.String("api-key", "", "ключ шлюза (лучше через env MENTOR_API_KEY)")
	direction := fs.String("direction", "", "направление: go|postgresql|algorithms|system-design|microservices")
	topic := fs.String("topic", "", "id или название темы")
	mode := fs.String("mode", "", "режим: learn|recall|interview|mistake|review|check|prod")
	list := fs.Bool("list", false, "показать направления и темы")
	due := fs.Bool("due", false, "показать темы к повторению")
	showProgress := fs.Bool("progress", false, "показать прогресс")
	demo := fs.Bool("demo", false, "офлайн-режим без сети (заглушка LLM)")
	showVersion := fs.Bool("version", false, "версия")
	_ = fs.Parse(os.Args[1:])

	if *showVersion {
		fmt.Printf("mentor %s\n", version)
		return
	}

	cfg, err := config.Load(*root, *baseURL, *apiKey, *model)
	if err != nil {
		fatal(err)
	}
	cur, err := curriculum.Load()
	if err != nil {
		fatal(err)
	}
	progJSON := filepath.Join(cfg.ProgressDir(), "progress.json")
	progMD := filepath.Join(cfg.ProgressDir(), "progress.md")
	store, err := progress.Load(progJSON)
	if err != nil {
		fatal(err)
	}

	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	out := os.Stdout

	// Отчёты без LLM.
	switch {
	case *list:
		printList(out, cfg.Color, cur, store)
		return
	case *due:
		printDue(out, cfg.Color, store)
		return
	case *showProgress:
		printProgress(out, cfg.Color, cur, store, progMD)
		return
	}

	// Сборка responder-а.
	var resp llm.Responder
	if *demo || !cfg.LLMReady() {
		if !*demo {
			fmt.Fprintf(out, "%s\n", ui.Yellow(cfg.Color, "LLM не настроен ("+cfg.LLMHint()+"). Работаю в офлайн-режиме (--demo)."))
		}
		resp = llm.Offline{}
	} else {
		resp = llm.New(cfg.BaseURL, cfg.APIKey, cfg.Model)
	}

	sess := &mentor.Session{
		Cur: cur, Store: store, Resp: resp, In: sc, Out: out,
		Color: cfg.Color, Lang: cfg.Language,
		ProgJSON: progJSON, ProgMD: progMD, SessionsDir: cfg.SessionsDir(),
	}
	ctx := context.Background()

	// Прямой запуск по флагам.
	if *direction != "" && *topic != "" && *mode != "" {
		if _, err := sess.Run(ctx, *direction, *topic, *mode); err != nil {
			fatal(err)
		}
		return
	}

	// Интерактивное меню.
	for {
		dir, ok := chooseDirection(out, sc, cfg.Color, cur)
		if !ok {
			return
		}
		topic, ok := chooseTopic(out, sc, cfg.Color, dir, store)
		if !ok {
			return
		}
		mode, ok := chooseMode(out, sc, cfg.Color)
		if !ok {
			return
		}
		if _, err := sess.Run(ctx, dir.ID, topic.ID, mode.ID); err != nil {
			fmt.Fprintf(out, "%s\n", ui.Red(cfg.Color, err.Error()))
		}
	}
}

func chooseDirection(out io.Writer, sc *bufio.Scanner, color bool, cur *curriculum.Curriculum) (curriculum.Direction, bool) {
	opts := make([]string, len(cur.Directions))
	for i, d := range cur.Directions {
		opts[i] = fmt.Sprintf("%s — %d тем", d.Title, len(d.Topics))
	}
	idx := ui.Menu(out, sc, color, "Направление:", opts)
	if idx < 0 {
		return curriculum.Direction{}, false
	}
	return cur.Directions[idx], true
}

func chooseTopic(out io.Writer, sc *bufio.Scanner, color bool, dir curriculum.Direction, store *progress.Store) (curriculum.Topic, bool) {
	opts := make([]string, len(dir.Topics))
	for i, t := range dir.Topics {
		m := 0
		if r := store.Get(dir.ID, t.ID); r != nil {
			m = r.Mastery
		}
		opts[i] = fmt.Sprintf("%s  %s", t.Title, ui.Dim(color, fmt.Sprintf("[%d/5]", m)))
	}
	idx := ui.Menu(out, sc, color, "Тема ("+dir.Title+"):", opts)
	if idx < 0 {
		return curriculum.Topic{}, false
	}
	return dir.Topics[idx], true
}

func chooseMode(out io.Writer, sc *bufio.Scanner, color bool) (mentor.Mode, bool) {
	opts := make([]string, len(mentor.Modes))
	for i, m := range mentor.Modes {
		opts[i] = fmt.Sprintf("%s — %s", m.Title, ui.Dim(color, m.Hint))
	}
	idx := ui.Menu(out, sc, color, "Режим:", opts)
	if idx < 0 {
		return mentor.Mode{}, false
	}
	return mentor.Modes[idx], true
}

func printList(out io.Writer, color bool, cur *curriculum.Curriculum, store *progress.Store) {
	for _, d := range cur.Directions {
		fmt.Fprintf(out, "\n%s\n", ui.Bold(color, d.Title+" ("+d.ID+")"))
		for _, t := range d.Topics {
			m := 0
			if r := store.Get(d.ID, t.ID); r != nil {
				m = r.Mastery
			}
			fmt.Fprintf(out, "  %-16s %-48s %s\n", t.ID, t.Title, ui.Dim(color, fmt.Sprintf("[%d/5]", m)))
		}
	}
}

func printDue(out io.Writer, color bool, store *progress.Store) {
	due := store.Due(time.Now())
	if len(due) == 0 {
		fmt.Fprintln(out, ui.Green(color, "Нет тем к повторению — начни новую тему."))
		return
	}
	fmt.Fprintf(out, "%s\n", ui.Bold(color, "Пора повторить:"))
	for _, d := range due {
		fmt.Fprintf(out, "  %s/%s  %s  %s\n", d.Direction, d.Topic,
			ui.Dim(color, fmt.Sprintf("[%d/5]", d.Record.Mastery)),
			ui.Dim(color, "следующее: "+d.Record.NextDue.Format("2006-01-02 15:04")))
	}
}

func printProgress(out io.Writer, color bool, cur *curriculum.Curriculum, store *progress.Store, progMD string) {
	fmt.Fprint(out, store.RenderMD(cur))
	if _, err := os.Stat(progMD); err == nil {
		fmt.Fprintf(out, "%s\n", ui.Dim(color, "файл: "+progMD))
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "mentor: %v\n", err)
	os.Exit(1)
}
