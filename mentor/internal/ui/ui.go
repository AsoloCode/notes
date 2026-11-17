// Package ui — минимальные цветные хелперы и меню для терминала (stdlib, без зависимостей).
package ui

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	cReset  = "\033[0m"
	cBold   = "\033[1m"
	cDim    = "\033[2m"
	cRed    = "\033[31m"
	cGreen  = "\033[32m"
	cYellow = "\033[33m"
	cCyan   = "\033[36m"
	cMag    = "\033[35m"
)

func wrap(on bool, code, s string) string {
	if !on {
		return s
	}
	return code + s + cReset
}

func Bold(on bool, s string) string   { return wrap(on, cBold, s) }
func Dim(on bool, s string) string    { return wrap(on, cDim, s) }
func Red(on bool, s string) string    { return wrap(on, cRed, s) }
func Green(on bool, s string) string  { return wrap(on, cGreen, s) }
func Yellow(on bool, s string) string { return wrap(on, cYellow, s) }
func Cyan(on bool, s string) string   { return wrap(on, cCyan, s) }
func Mag(on bool, s string) string    { return wrap(on, cMag, s) }

// Menu печатает нумерованный список и возвращает индекс выбранного пункта (0-based).
// Возвращает -1, если ввод закончился (EOF) или введён неверный номер трижды.
func Menu(out io.Writer, sc *bufio.Scanner, color bool, title string, options []string) int {
	fmt.Fprintf(out, "\n%s\n", Bold(color, title))
	for i, o := range options {
		fmt.Fprintf(out, "  %s %s\n", Cyan(color, fmt.Sprintf("%d)", i+1)), o)
	}
	if len(options) == 0 {
		return -1
	}
	for attempt := 0; attempt < 3; attempt++ {
		fmt.Fprintf(out, "%s ", Yellow(color, ">"))
		if !sc.Scan() {
			return -1
		}
		txt := strings.TrimSpace(sc.Text())
		n, err := strconv.Atoi(txt)
		if err == nil && n >= 1 && n <= len(options) {
			return n - 1
		}
		fmt.Fprintf(out, "%s\n", Dim(color, "Введи номер из списка."))
	}
	return -1
}
