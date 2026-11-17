package validate

import (
	"path/filepath"
	"testing"
)

func TestBankIsValid(t *testing.T) {
	problems := ValidateBank(filepath.Join("..", ".."))
	if len(problems) != 0 {
		t.Fatalf("ожидался валидный банк, проблемы: %v", problems)
	}
}
