package mentor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AsoloCode/notes/mentor/internal/curriculum"
	"github.com/AsoloCode/notes/mentor/internal/llm"
	"github.com/AsoloCode/notes/mentor/internal/progress"
	"github.com/AsoloCode/notes/mentor/internal/ui"
)

// Session — одна интерактивная сессия ментора.
type Session struct {
	Cur         *curriculum.Curriculum
	Store       *progress.Store
	Resp        llm.Responder
	In          *bufio.Scanner
	Out         io.Writer
	Color       bool
	Lang        string
	ProgJSON    string
	ProgMD      string
	SessionsDir string
	Now         func() time.Time // инъекция времени (тесты)
}

func (s *Session) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Session) printf(format string, a ...any) {
	fmt.Fprintf(s.Out, format, a...)
}

// Run проигрывает сессию и сохраняет прогресс. Возвращает число оценённых ответов.
func (s *Session) Run(ctx context.Context, dirID, topicID, modeID string) (int, error) {
	dir, ok := s.Cur.Direction(dirID)
	if !ok {
		return 0, fmt.Errorf("направление %q не найдено", dirID)
	}
	topic, ok := dir.Topic(topicID)
	if !ok {
		return 0, fmt.Errorf("тема %q не найдена в направлении %q", topicID, dir.Title)
	}
	mode, ok := ModeByID(modeID)
	if !ok {
		return 0, fmt.Errorf("режим %q не найден", modeID)
	}

	rec := s.Store.Get(dir.ID, topic.ID)
	mastery := 0
	if rec != nil {
		mastery = rec.Mastery
	}

	msgs := []llm.Message{{Role: "system", Content: buildSystemPrompt(dir, topic, mode, mastery, s.Lang)}}

	s.printf("\n%s\n", ui.Bold(s.Color, fmt.Sprintf("=== %s · %s · %s ===", dir.Title, topic.Title, mode.Title)))
	s.printf("%s\n", ui.Dim(s.Color, "Команды: /done — закончить и сохранить · /skip — пропустить · /quit — выйти · Ctrl+C — прервать"))
	if strings.TrimSpace(topic.Material) != "" {
		s.printf("%s\n", ui.Dim(s.Color, "Материал: "+topic.Material))
	}
	s.printf("\n")

	// Открывающее сообщение ментора.
	opening := llm.Message{Role: "user", Content: openingInstruction(mode)}
	turnMsgs := append(append([]llm.Message{}, msgs...), opening)
	reply, err := s.Resp.Respond(ctx, turnMsgs)
	if err != nil {
		s.printf("%s\n", ui.Red(s.Color, "ментор недоступен: "+err.Error()))
	} else {
		clean, _, _ := parseScore(reply)
		msgs = append(msgs, llm.Message{Role: "assistant", Content: clean})
		s.printf("%s %s\n\n", ui.Cyan(s.Color, "Ментор:"), clean)
	}

	scored := 0
	for {
		if ctx.Err() != nil {
			return scored, ctx.Err()
		}
		s.printf("%s ", ui.Yellow(s.Color, "Вы:"))
		if !s.In.Scan() {
			break // EOF
		}
		line := strings.TrimSpace(s.In.Text())
		if line == "" {
			continue
		}
		switch strings.ToLower(line) {
		case "/quit", "/exit", "/q":
			_ = s.save(ctx, dir, topic, mode, msgs, scored)
			return scored, nil
		case "/done":
			_ = s.save(ctx, dir, topic, mode, msgs, scored)
			s.printf("\n%s\n", ui.Green(s.Color, "Сессия сохранена."))
			return scored, nil
		case "/help":
			s.printf("%s\n", ui.Dim(s.Color, "Отвечай своими словами. /done — сохранить, /skip — пропустить вопрос, /quit — выйти."))
			continue
		case "/skip":
			line = "Пропустим этот вопрос. Задай другой, пожалуйста."
		}

		msgs = append(msgs, llm.Message{Role: "user", Content: line})
		reply, err := s.Resp.Respond(ctx, msgs)
		if err != nil {
			s.printf("%s\n\n", ui.Red(s.Color, "ошибка ментора: "+err.Error()))
			msgs = msgs[:len(msgs)-1] // откатываем неудачный ход
			continue
		}
		clean, score, ok := parseScore(reply)
		msgs = append(msgs, llm.Message{Role: "assistant", Content: clean})
		s.printf("%s %s\n", ui.Cyan(s.Color, "Ментор:"), clean)
		if ok && mode.Evaluating {
			scored++
			s.Store.Update(dir.ID, topic.ID, mode.ID, score, "", s.now())
			s.printf("%s\n", ui.Dim(s.Color, fmt.Sprintf("оценка: %d/5", score)))
		}
		s.printf("\n")
	}
	_ = s.save(ctx, dir, topic, mode, msgs, scored)
	return scored, nil
}

// save пишет журнал сессии и обновляет файлы прогресса.
func (s *Session) save(_ context.Context, dir curriculum.Direction, topic curriculum.Topic, mode Mode, msgs []llm.Message, scored int) error {
	if err := os.MkdirAll(s.SessionsDir, 0o755); err != nil {
		return err
	}
	r := s.Store.Get(dir.ID, topic.ID)
	mlvl := 0
	if r != nil {
		mlvl = r.Mastery
	}
	name := fmt.Sprintf("%s_%s_%s-%s.md", s.now().Format("2006-01-02_1504"), mode.ID, dir.ID, topic.ID)
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("type: session\n")
	b.WriteString(fmt.Sprintf("direction: %s\n", dir.ID))
	b.WriteString(fmt.Sprintf("topic: %s\n", topic.ID))
	b.WriteString(fmt.Sprintf("mode: %s\n", mode.ID))
	b.WriteString(fmt.Sprintf("date: %s\n", s.now().Format("2006-01-02 15:04")))
	b.WriteString(fmt.Sprintf("scored_answers: %d\n", scored))
	b.WriteString(fmt.Sprintf("mastery_after: %d\n", mlvl))
	b.WriteString("---\n\n")
	b.WriteString(fmt.Sprintf("# %s · %s · %s\n\n", dir.Title, topic.Title, mode.Title))
	for _, m := range msgs {
		switch m.Role {
		case "assistant":
			b.WriteString("**Ментор:** " + m.Content + "\n\n")
		case "user":
			b.WriteString("**Ученик:** " + m.Content + "\n\n")
		}
	}
	if err := os.WriteFile(filepath.Join(s.SessionsDir, name), []byte(b.String()), 0o644); err != nil {
		return err
	}

	if err := s.Store.SaveJSON(s.ProgJSON); err != nil {
		return err
	}
	return os.WriteFile(s.ProgMD, []byte(s.Store.RenderMD(s.Cur)), 0o644)
}
