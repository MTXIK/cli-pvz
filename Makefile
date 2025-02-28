# Имя исполняемого файла
OUTPUT := PVZ

# Точка входа (путь к файлу main.go)
MAIN := cmd/main.go

# Порог сложности для линтеров
COMPLEXITY_THRESHOLD := 10

# Путь к установленным Go-бинарникам
GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin

.PHONY: build run clean fmt lint install-linters

# Установка линтеров
install-linters:
	@echo "Установка линтеров..."
	@go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	@go install github.com/uudashr/gocognit/cmd/gocognit@latest

# Проверка линтерами
lint: install-linters
	@echo "\n"
	@echo "Проверка циклометрической сложности (gocyclo)..."
	@$(GOBIN)/gocyclo -over $(COMPLEXITY_THRESHOLD) . || true
	@echo "Проверка когнитивной сложности (gocognit)..."
	@$(GOBIN)/gocognit -over $(COMPLEXITY_THRESHOLD) . || true
	@echo "\n"

# Цель сборки: сначала линтинг и форматирование, затем создаём исполняемый файл
build: lint fmt
	@echo "Сборка проекта..."
	@go build -o $(OUTPUT) $(MAIN)

# Цель запуска: сначала сборка, затем запуск исполняемого файла
run: build
	@echo "Запуск проекта..."
	@./$(OUTPUT)

# Цель очистки: удаляет скомпилированный исполняемый файл
clean:
	@echo "Очистка проекта..."
	@rm -f $(OUTPUT)

# Форматирование кода
fmt:
	@echo "Форматирование кода..."
	@go fmt ./...