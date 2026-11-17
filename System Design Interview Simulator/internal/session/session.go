// Package session реализует команды CLI симулятора.
package session

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AsoloCode/notes/sdsim/internal/bank"
	"github.com/AsoloCode/notes/sdsim/internal/rubric"
	"github.com/AsoloCode/notes/sdsim/internal/validate"
)

const prog = "sdsim"

// Run выполняет команду и возвращает код выхода.
func Run(args []string) int {
	if len(args) == 0 {
		usage()
		return 1
	}
	switch args[0] {
	case "list":
		return cmdList(args[1:])
	case "start":
		return cmdStart(args[1:])
	case "submit":
		return cmdSubmit(args[1:])
	case "reveal":
		return cmdReveal(args[1:])
	case "score":
		return cmdScore(args[1:])
	case "validate":
		return cmdValidate(args[1:])
	case "-h", "--help", "help":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "неизвестная команда: %s\n", args[0])
		usage()
		return 1
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `System Design Interview Simulator

  sdsim list
  sdsim start <slug>
  sdsim submit <slug> --file <path>
  sdsim reveal <slug> [--force]
  sdsim score <slug> --scores 3,4,2,3,3,3 [--candidate NAME] [--notes TEXT]
  sdsim validate
`)
}

// ---------- list ----------

func cmdList(args []string) int {
	root, _, code := setup("list", args)
	if code >= 0 {
		return code
	}
	b, err := bank.Load(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("%2s  %-22s %-12s %s\n", "#", "slug", "level", "title")
	fmt.Println(strings.Repeat("-", 70))
	for _, sc := range b.Scenarios {
		fmt.Printf("%2d  %-22s %-12s %s\n", sc.Order, sc.Slug, sc.Level, sc.Title)
	}
	fmt.Printf("\nВсего: %d. Старт: %s start <slug>\n", len(b.Scenarios), prog)
	return 0
}

// ---------- start ----------

func cmdStart(args []string) int {
	root, rest, code := setup("start", args)
	if code >= 0 {
		return code
	}
	slug, _, ok := splitSlug(rest)
	if !ok {
		fmt.Fprintf(os.Stderr, "использование: %s start <slug>\n", prog)
		return 1
	}
	b, sc, code := loadScenario(root, slug)
	if code != 0 {
		return code
	}
	_ = b
	card, err := os.ReadFile(filepath.Join(root, sc.ScenarioPath))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Print(string(card))
	fmt.Println("\n--- Вопросы интервьюера (для самопроверки) ---")
	for i, q := range sc.InterviewerQuestions {
		fmt.Printf("%d. %s\n", i+1, q)
	}
	fmt.Printf("\nКогда зафиксируете ответ: %s submit %s --file <ваш файл>\n", prog, sc.Slug)
	fmt.Printf("Затем: %s reveal %s\n", prog, sc.Slug)
	return 0
}

// ---------- submit ----------

func cmdSubmit(args []string) int {
	root, rest, code := setup("submit", args)
	if code >= 0 {
		return code
	}
	slug, flags, ok := splitSlug(rest, "file")
	if !ok {
		fmt.Fprintf(os.Stderr, "использование: %s submit <slug> --file <path>\n", prog)
		return 1
	}
	fs := flag.NewFlagSet("submit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	file := fs.String("file", "", "файл с вашим ответом")
	if err := fs.Parse(flags); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка флагов: %v\n", err)
		return 1
	}
	if *file == "" {
		fmt.Fprintf(os.Stderr, "использование: %s submit <slug> --file <path>\n", prog)
		return 1
	}
	_, sc, code := loadScenario(root, slug)
	if code != 0 {
		return code
	}
	src, err := os.ReadFile(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "файл попытки не найден: %s\n", *file)
		return 1
	}
	outDir := filepath.Join(root, "attempts")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	dst := filepath.Join(outDir, fmt.Sprintf("%02d-%s-%s.md", sc.Order, sc.Slug, today()))
	if err := os.WriteFile(dst, src, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	rel, _ := filepath.Rel(root, dst)
	fmt.Printf("Попытка сохранена: %s\n", rel)
	fmt.Printf("Теперь можно открыть эталон: %s reveal %s\n", prog, sc.Slug)
	return 0
}

// ---------- reveal ----------

func cmdReveal(args []string) int {
	root, rest, code := setup("reveal", args)
	if code >= 0 {
		return code
	}
	slug, flags, ok := splitSlug(rest)
	if !ok {
		fmt.Fprintf(os.Stderr, "использование: %s reveal <slug> [--force]\n", prog)
		return 1
	}
	fs := flag.NewFlagSet("reveal", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	force := fs.Bool("force", false, "обойти гейт (не рекомендуется)")
	if err := fs.Parse(flags); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка флагов: %v\n", err)
		return 1
	}
	_, sc, code := loadScenario(root, slug)
	if code != 0 {
		return code
	}
	if len(bank.AttemptFiles(root, slug)) == 0 && !*force {
		fmt.Fprintf(os.Stderr,
			"ГЕЙТ: попытка не зафиксирована, эталон скрыт.\n"+
				"Сначала: %s start %s  →  %s submit %s --file <файл>\n"+
				"Осознанный обход (не рекомендуется): reveal --force\n", prog, slug, prog, slug)
		return 2
	}
	sol, err := os.ReadFile(filepath.Join(root, sc.SolutionPath))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Print(string(sol))
	fmt.Println("\n--- Red flags сценария ---")
	for _, rf := range sc.RedFlags {
		fmt.Printf("  ✗ %s\n", rf)
	}
	fmt.Println("--- Strong signals ---")
	for _, ss := range sc.StrongSignals {
		fmt.Printf("  ✓ %s\n", ss)
	}
	return 0
}

// ---------- score ----------

func cmdScore(args []string) int {
	root, rest, code := setup("score", args)
	if code >= 0 {
		return code
	}
	slug, flags, ok := splitSlug(rest, "scores", "candidate", "notes")
	if !ok {
		fmt.Fprintf(os.Stderr, "использование: %s score <slug> --scores 3,4,2,3,3,3\n", prog)
		return 1
	}
	fs := flag.NewFlagSet("score", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scoresRaw := fs.String("scores", "", "оценки через запятую, напр. 3,4,2,3,3,3")
	candidate := fs.String("candidate", "candidate", "имя кандидата")
	notes := fs.String("notes", "", "заметки")
	if err := fs.Parse(flags); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка флагов: %v\n", err)
		return 1
	}
	if *scoresRaw == "" {
		fmt.Fprintf(os.Stderr, "использование: %s score <slug> --scores 3,4,2,3,3,3\n", prog)
		return 1
	}
	_, sc, code := loadScenario(root, slug)
	if code != 0 {
		return code
	}
	card, err := rubric.Load(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	scores, err := parseScores(*scoresRaw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if len(scores) != len(card.Dimensions) {
		fmt.Fprintf(os.Stderr, "ожидалось %d оценок (по измерениям), получено %d.\nПорядок измерений:\n", len(card.Dimensions), len(scores))
		for i, d := range card.Dimensions {
			fmt.Fprintf(os.Stderr, "  %d. %s (%s, вес %d)\n", i+1, d.ID, d.Title, d.Weight)
		}
		return 1
	}
	for _, s := range scores {
		if s < card.Scale.Min || s > card.Scale.Max {
			fmt.Fprintf(os.Stderr, "оценки должны быть в диапазоне %d..%d\n", card.Scale.Min, card.Scale.Max)
			return 1
		}
	}

	total, grade, results := card.Total(scores)
	fmt.Printf("%-28s%7s%6s%8s\n", "измерение", "оценка", "вес", "вклад")
	fmt.Println(strings.Repeat("-", 52))
	for _, r := range results {
		fmt.Printf("%-28s%5s%6d%8.1f\n",
			r.Dimension.ID,
			fmt.Sprintf("%d/%d", r.Score, card.Scale.Max),
			r.Dimension.Weight,
			r.Contribution)
	}
	fmt.Println(strings.Repeat("-", 52))
	fmt.Printf("%-28s%7s%6s%8.1f  (%s)\n", "ИТОГ", "", "", total, grade)

	scoresMap := make(map[string]int, len(card.Dimensions))
	for i, d := range card.Dimensions {
		scoresMap[d.ID] = scores[i]
	}
	record := struct {
		Slug      string         `json:"scenario_slug"`
		Candidate string         `json:"candidate"`
		Date      string         `json:"date"`
		Scores    map[string]int `json:"scores"`
		Total     float64        `json:"total"`
		Grade     string         `json:"grade"`
		Notes     string         `json:"notes"`
	}{sc.Slug, *candidate, today(), scoresMap, total, grade, *notes}

	outDir := filepath.Join(root, "attempts")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	dst := filepath.Join(outDir, fmt.Sprintf("%02d-%s-%s.score.json", sc.Order, sc.Slug, today()))
	raw, _ := json.MarshalIndent(record, "", "  ")
	raw = append(raw, '\n')
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	rel, _ := filepath.Rel(root, dst)
	fmt.Printf("Оценочный лист сохранён: %s\n", rel)
	return 0
}

// ---------- validate ----------

func cmdValidate(args []string) int {
	root, _, code := setup("validate", args)
	if code >= 0 {
		return code
	}
	problems := validate.ValidateBank(root)
	if len(problems) > 0 {
		fmt.Println("НАЙДЕНЫ ПРОБЛЕМЫ:")
		for _, p := range problems {
			fmt.Printf("  - %s\n", p)
		}
		return 1
	}
	fmt.Println("OK: банк сценариев и оценочный лист валидны.")
	return 0
}

// ---------- helpers ----------

// setup извлекает общий флаг --root (в любой позиции) и находит каталог симулятора.
// Код возврата >= 0 означает, что команду нужно завершить этим кодом.
func setup(name string, args []string) (string, []string, int) {
	rootFlag, remaining := extractRoot(args)
	root, err := findRoot(rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", nil, 1
	}
	return root, remaining, -1
}

// extractRoot вынимает --root VALUE / --root=VALUE из args.
func extractRoot(args []string) (string, []string) {
	var root string
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--root":
			if i+1 < len(args) {
				root = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--root="):
			root = strings.TrimPrefix(a, "--root=")
		default:
			out = append(out, a)
		}
	}
	return root, out
}

// splitSlug отделяет первый позиционный slug от флагов, зная, какие флаги берут значение.
func splitSlug(args []string, valuedFlags ...string) (slug string, flags []string, ok bool) {
	valued := make(map[string]bool, len(valuedFlags))
	for _, f := range valuedFlags {
		valued[f] = true
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			name := strings.TrimLeft(a, "-")
			if strings.ContainsRune(name, '=') {
				continue
			}
			if valued[name] {
				i++
			}
			continue
		}
		flags = append(append([]string{}, args[:i]...), args[i+1:]...)
		return a, flags, true
	}
	return "", args, false
}

func loadScenario(root, slug string) (*bank.Bank, *bank.Scenario, int) {
	b, err := bank.Load(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, nil, 1
	}
	sc, ok := b.Find(slug)
	if !ok {
		fmt.Fprintf(os.Stderr, "неизвестный сценарий: %s\n", slug)
		return nil, nil, 1
	}
	return b, sc, 0
}

func parseScores(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, errors.New("--scores должен быть списком целых через запятую, напр. 3,4,2,3,3,3")
		}
		out = append(out, n)
	}
	return out, nil
}

func today() string {
	return time.Now().Format("2006-01-02")
}

// findRoot ищет каталог симулятора (с data/scenarios.json) вверх от CWD,
// затем от каталога исполняемого файла.
func findRoot(explicit string) (string, error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", err
		}
		if isRoot(abs) {
			return abs, nil
		}
		return "", fmt.Errorf("в '%s' нет data/scenarios.json", abs)
	}
	if cwd, err := os.Getwd(); err == nil {
		if r, ok := searchUp(cwd); ok {
			return r, nil
		}
	}
	if exe, err := os.Executable(); err == nil {
		if r, ok := searchUp(filepath.Dir(exe)); ok {
			return r, nil
		}
	}
	return "", errors.New("не найден каталог симулятора (data/scenarios.json). Укажите --root")
}

func searchUp(start string) (string, bool) {
	dir := start
	for i := 0; i < 8; i++ {
		if isRoot(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func isRoot(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "data", "scenarios.json"))
	return err == nil
}
