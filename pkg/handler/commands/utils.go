package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"gitlab.ozon.dev/gojhw1/pkg/model"
	"golang.org/x/term"
)

type acceptOrderParams struct {
	orderID     int64
	clientID    int64
	deadline    time.Time
	weight      float64
	cost        float64
	packageType *model.PackageType
	wrapper     *model.WrapperType
}

func validateAcceptOrderArgs(args []string) error {
	if len(args) < 5 {
		return ErrInvalidAcceptOrderArgs
	}
	return nil
}

func parseAcceptOrderParams(args []string) (*acceptOrderParams, error) {
	orderID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("неверный формат orderID: %v", err)
	}

	clientID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("неверный формат clientID: %v", err)
	}

	deadline, err := parseDeadline(args[2])
	if err != nil {
		return nil, err
	}

	weight, err := strconv.ParseFloat(args[3], 64)
	if err != nil {
		return nil, fmt.Errorf("неверный формат веса: %v", err)
	}
	if weight <= 0 {
		return nil, errors.New("вес должен быть больше 0")
	}

	cost, err := strconv.ParseFloat(args[4], 64)
	if err != nil {
		return nil, fmt.Errorf("неверный формат стоимости: %v", err)
	}
	if cost <= 0 {
		return nil, errors.New("стоимость должна быть больше 0")
	}

	packageType, wrapper := parsePackageInfo(args)

	return &acceptOrderParams{
		orderID:     orderID,
		clientID:    clientID,
		deadline:    deadline,
		weight:      weight,
		cost:        cost,
		packageType: packageType,
		wrapper:     wrapper,
	}, nil
}

func parseDeadline(deadlineStr string) (time.Time, error) {
	if dur, err := time.ParseDuration(deadlineStr); err == nil {
		return time.Now().Add(dur), nil
	}

	deadline, err := time.Parse(timeLayout, deadlineStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("неверный формат даты или длительности: %v", err)
	}
	return deadline, nil
}

func parsePackageInfo(args []string) (*model.PackageType, *model.WrapperType) {
	if len(args) <= 5 {
		return nil, nil
	}

	parts := strings.Split(args[5], "+")
	pt := model.PackageType(strings.TrimSpace(parts[0]))
	packageType := &pt

	var wrapper *model.WrapperType
	if len(parts) > 1 {
		wt := model.WrapperType(strings.TrimSpace(parts[1]))
		wrapper = &wt
	}

	return packageType, wrapper
}

func listReturnsPrintFull(returns []model.Order, pageSize int) error {
	totalReturns := len(returns)
	totalPages := (totalReturns + pageSize - 1) / pageSize
	reader := bufio.NewReader(os.Stdin)

	for i := range totalPages {
		start := i * pageSize
		end := min(start+pageSize, totalReturns)

		if err := listReturnsPrintOrders(returns, start, end); err != nil {
			return err
		}

		fmt.Printf("Страница %d из %d (Всего возвратов: %d)\n", i+1, totalPages, totalReturns)

		if i < totalPages-1 {
			fmt.Print("Нажмите Enter для следующей страницы...")
			_, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("ошибка при чтении ввода: %v", err)
			}
		}
	}

	return nil
}

func listReturnsPrintOrders(returns []model.Order, start, end int) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "ID\tКлиент\tВремя возврата"); err != nil {
		return fmt.Errorf("ошибка при записи заголовка: %v", err)
	}

	for _, order := range returns[start:end] {
		ret := "-"
		if order.ReturnedAt != nil {
			ret = order.ReturnedAt.Format(timeLayout)
		}
		if _, err := fmt.Fprintf(w, "%d\t%d\t%s\n",
			order.ID,
			order.CustomerID,
			ret); err != nil {
			return fmt.Errorf("ошибка при записи данных: %v", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("ошибка при выводе таблицы: %v", err)
	}

	return nil
}

type listOrdersParams struct {
	customerID int64
	lastN      int
	filterPVZ  bool
	pageSize   int
}

func parseListOrdersParams(args []string) (*listOrdersParams, error) {
	if len(args) < 1 {
		return nil, ErrInvalidListOrdersArgs
	}

	customerID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("неверный формат customerID: %v", err)
	}

	params := &listOrdersParams{
		customerID: customerID,
		pageSize:   defaultPageSize,
	}

	for i := 1; i < len(args); i++ {
		nextIndex, err := processListOrderArg(args, i, params)
		if err != nil {
			return nil, err
		}
		i = nextIndex
	}

	if params.pageSize <= 0 {
		return nil, ErrInvalidPageSize
	}

	return params, nil
}

func processListOrderArg(args []string, i int, params *listOrdersParams) (int, error) {
	switch args[i] {
	case "last":
		nextIndex, err := processLOLast(args, i, params)
		if err != nil {
			return 0, err
		}
		i = nextIndex
	case "pvz":
		params.filterPVZ = true
	case "pageSize":
		nextIndex, err := processLOPageSize(args, i, params)
		if err != nil {
			return 0, err
		}
		i = nextIndex
	}
	return i, nil
}

func processLOLast(args []string, i int, params *listOrdersParams) (int, error) {
	if i+1 < len(args) {
		n, err := strconv.Atoi(args[i+1])
		if err != nil {
			return 0, fmt.Errorf("%v - после last должно быть число, получено: %s", err, args[i+1])
		}
		params.lastN = n
		return i + 1, nil
	}
	return i, nil
}

func processLOPageSize(args []string, i int, params *listOrdersParams) (int, error) {
	if i+1 < len(args) {
		size, err := strconv.Atoi(args[i+1])
		if err != nil {
			return 0, fmt.Errorf("%v - после pageSize должно быть число, получено: %s", err, args[i+1])
		}
		params.pageSize = size
		return i + 1, nil
	}
	return i, nil
}

func displayLOOrders(h *Handler, terminal *term.Terminal, ordersList []model.Order, currentPos int, totalOrders, pageSize int) error {
	h.clearTerminal()

	w := tabwriter.NewWriter(terminal, 0, 0, 2, ' ', 0)

	if _, err := fmt.Fprintln(terminal, "\t\t\t\t=== Список заказов ==="); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "ID\tКлиент\tСрок хранения\tСостояние\tЦена\tВес\tУпаковка\tОбновлен"); err != nil {
		return err
	}

	end := min(currentPos+pageSize, totalOrders)

	for i := currentPos; i < end; i++ {
		order := ordersList[i]

		if _, err := fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%.2f\t%.2f\t%s\t%s\n",
			order.ID,
			order.CustomerID,
			order.DeadlineAt.Format(timeLayout),
			order.State,
			order.Cost,
			order.Weight,
			formatPackageInfo(order),
			order.UpdatedAt.Format(timeLayout)); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}

	if err := displayLOControls(terminal, currentPos, totalOrders, pageSize, end); err != nil {
		return err
	}

	return displayLOProgressBar(terminal, currentPos, pageSize, totalOrders)
}

func displayLOControls(terminal *term.Terminal, currentPos int, totalOrders, pageSize, end int) error {
	if _, err := fmt.Fprintf(terminal, "\nПоказано %d-%d из %d заказов\n", currentPos+1, end, totalOrders); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(terminal, "\nУправление:"); err != nil {
		return err
	}
	if totalOrders > pageSize {
		if _, err := fmt.Fprintln(terminal, "↑ - Прокрутка вверх"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(terminal, "↓ - Прокрутка вниз"); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(terminal, "q - Выход"); err != nil {
		return err
	}

	return nil
}

func displayLOProgressBar(terminal *term.Terminal, currentPos, pageSize, totalOrders int) error {
	if totalOrders > pageSize {
		progress := float64(currentPos) / float64(totalOrders-pageSize)
		barWidth := 25
		position := int(progress * float64(barWidth))

		bar := strings.Repeat("-", barWidth)
		if _, err := fmt.Fprintf(terminal, "\nПрогресс: [%s%s%s]",
			bar[:position],
			"●",
			bar[position:]); err != nil {
			return err
		}
	}

	return nil
}

func handleLOKeyPress(displayOrders func() error, currentPos *int, totalOrders, pageSize int) error {
	for {
		// Чтение управляющих клавиш
		bytes := make([]byte, 3)
		_, err := os.Stdin.Read(bytes)
		if err != nil {
			return fmt.Errorf("ошибка при чтении ввода: %v", err)
		}

		switch {
		case bytes[0] == 'q' || bytes[0] == 'Q':
			return nil
		case bytes[0] == 27 && bytes[1] == 91: // Escape последовательности для стрелок
			if err = processLOArrowKey(bytes[2], displayOrders, currentPos, totalOrders, pageSize); err != nil {
				return err
			}
		}
	}
}

func processLOArrowKey(key byte, displayOrders func() error, currentPos *int, totalOrders, pageSize int) error {
	switch key {
	case 65: // Стрелка вверх
		return processLOUpArrow(displayOrders, currentPos)
	case 66: // Стрелка вниз
		return processLODownArrow(displayOrders, currentPos, totalOrders, pageSize)
	}

	return nil
}

func processLOUpArrow(displayOrders func() error, currentPos *int) error {
	if *currentPos > 0 {
		*currentPos-- // Прокрутка на одну строку вверх
		return displayOrders()
	}

	return nil
}

func processLODownArrow(displayOrders func() error, currentPos *int, totalOrders, pageSize int) error {
	if *currentPos < totalOrders-pageSize {
		*currentPos++ // Прокрутка на одну строку вниз
		return displayOrders()
	}

	return nil
}

func formatPackageInfo(order model.Order) string {
	if order.PackageType == nil {
		return "-"
	}
	result := string(*order.PackageType)
	if order.Wrapper != nil {
		result += " + " + string(*order.Wrapper)
	}

	return result
}
