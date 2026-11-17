// Command sdsim — CLI-раннер симулятора System Design интервью.
//
// Реализует гейт «эталон — только после попытки»: команда reveal откажется
// показывать решение, если для сценария нет зафиксированной попытки
// (обойти можно флагом --force, но это осознанное нарушение правил тренировки).
//
// Команды:
//
//	list                              — список сценариев
//	start <slug>                      — показать бриф и вопросы (без решения)
//	submit <slug> --file <path>       — сохранить попытку
//	reveal <slug> [--force]           — показать эталон (после попытки)
//	score <slug> --scores 3,4,2,3,3,3 — посчитать балл и грейд
//	validate                          — валидация банка сценариев
package main

import (
	"os"

	"github.com/AsoloCode/notes/sdsim/internal/session"
)

func main() {
	os.Exit(session.Run(os.Args[1:]))
}
