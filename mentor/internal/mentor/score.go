package mentor

import (
	"regexp"
	"strconv"
	"strings"
)

var scoreRe = regexp.MustCompile(`\[\[\s*SCORE\s*:\s*(none|\d+)\s*\]\]`)

// parseScore вытаскивает метку [[SCORE:n]] из ответа модели и возвращает:
//   - текст без метки,
//   - score (0..5),
//   - ok=false, если метки нет или она [[SCORE:none]].
func parseScore(text string) (clean string, score int, ok bool) {
	loc := scoreRe.FindStringSubmatchIndex(text)
	if loc == nil {
		return text, 0, false
	}
	token := text[loc[0]:loc[1]]
	val := scoreRe.FindStringSubmatch(text)[1]
	clean = strings.TrimSpace(strings.Replace(text, token, "", 1))
	clean = strings.TrimRight(clean, " \t\r\n")
	if strings.EqualFold(val, "none") {
		return clean, 0, false
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return clean, 0, false
	}
	if n < 0 {
		n = 0
	}
	if n > 5 {
		n = 5
	}
	return clean, n, true
}
