package llm

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientRespond(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"привет"}}]}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "secret", "model-x")
	got, err := c.Respond(context.Background(), []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "привет" {
		t.Fatalf("got %q", got)
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("path %q", gotPath)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("auth %q", gotAuth)
	}
}

func TestClientHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"boom"}}`, http.StatusUnauthorized)
	}))
	defer srv.Close()
	c := New(srv.URL, "k", "m")
	if _, err := c.Respond(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("ожидалась ошибка 401, got %v", err)
	}
}

func TestEndpointVariants(t *testing.T) {
	if got := New("http://x", "k", "m").endpoint(); got != "http://x/v1/chat/completions" {
		t.Fatalf("без /v1: %q", got)
	}
	if got := New("http://x/v1/", "k", "m").endpoint(); got != "http://x/v1/chat/completions" {
		t.Fatalf("с /v1: %q", got)
	}
}

func TestOfflineRespondsWithScore(t *testing.T) {
	off := Offline{}
	out, err := off.Respond(context.Background(), []Message{{Role: "user", Content: "мой ответ"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[[SCORE:") {
		t.Fatalf("офлайн-ответ без метки: %q", out)
	}
	if _, _, ok := parseScoreLocal(out); !ok {
		t.Fatalf("метка не распознаётся: %q", out)
	}
}

// parseScoreLocal — локальная проверка формата метки (без импорта пакета mentor).
func parseScoreLocal(s string) (string, int, bool) {
	i := strings.Index(s, "[[SCORE:")
	if i < 0 {
		return s, 0, false
	}
	return s, 3, true
}
