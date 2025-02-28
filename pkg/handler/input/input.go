package input

import (
	"strings"

	"github.com/chzyer/readline"
)

// Handler - оборачивает объект readline.Terminal.
type Handler struct {
	Terminal *readline.Instance
}

// NewHandler - инициализирует и возвращает новый Handler.
// Он настраивает prompt, interrupt и EOF сообщения.
func NewHandler() (*Handler, error) {
	inputHandler, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return nil, err
	}
	return &Handler{
		Terminal: inputHandler,
	}, nil
}

// Close - освобождает ресурсы, связанные с readline.
func (h *Handler) Close() {
	h.Terminal.Close()
}

// ReadLine - читает одну строку ввода пользователя и возвращает её.
func (h *Handler) ReadLine() (string, error) {
	return h.Terminal.Readline()
}

// ProcessLine - обрабатывает строку, полученную из ReadLine
func (h *Handler) ProcessLine(line string) (string, []string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", nil
	}

	parts := strings.Fields(line)
	command := parts[0]
	args := parts[1:]
	return command, args
}
