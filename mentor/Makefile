.PHONY: build test vet fmt run clean

BIN := bin/mentor

build: ## Собрать бинарь в bin/mentor
	@mkdir -p bin
	go build -o $(BIN) ./cmd/mentor
	@echo "built: $(BIN)"

test: ## Прогнать тесты
	go test ./...

vet: ## go vet
	go vet ./...

fmt: ## gofmt
	gofmt -w .

run: build ## Собрать и запустить меню
	./$(BIN)

clean: ## Удалить артефакты сборки
	rm -rf bin/

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-8s %s\n", $$1, $$2}'
