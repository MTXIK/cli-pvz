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
	"gitlab.ozon.dev/gojhw1/pkg/service"
	"gitlab.ozon.dev/gojhw1/pkg/storage"
	"golang.org/x/term"
)

var (
	ErrInvalidAcceptOrderArgs     = errors.New("использование: accept_order <orderID> <ClientID> <deadline> <weight> <cost> [package_type[+wrapper]]")
	ErrInvalidReturnCourierArgs   = errors.New("использование: return_to_courier <orderID>")
	ErrInvalidProcessCustomerArgs = errors.New("использование: process_customer <customerID> <action> <orderID1> [orderID2 ...]")
	ErrInvalidListOrdersArgs      = errors.New("использование: list_orders <customerID> [pageSize <N>][last <N>] [pvz]")
	ErrInvalidListReturnsArgs     = errors.New("использование: list_returns pageSize <size>")
	ErrInvalidAcceptFileArgs      = errors.New("использование: accept_orders_file <filename>")
	ErrInvalidPageSize            = errors.New("размер страницы должен быть больше 0")
)

const timeLayout = "2006-01-02T15:04:05"
const defaultPageSize = 5

type CommandFunc func([]string) error

type Handler struct {
	service  *service.OrderService
	storage  storage.OrderStorage
	commands map[string]CommandFunc
}

// NewHandler - Создает новый обработчик команд
func NewHandler(service *service.OrderService, storage storage.OrderStorage) *Handler {
	Handler := &Handler{
		service: service,
		storage: storage,
	}

	Handler.commands = map[string]CommandFunc{
		"help": func(_ []string) error {
			Handler.printHelp()
			return nil
		},
		"exit": func(_ []string) error {
			fmt.Println("Выход...")
			os.Exit(0)
			return nil
		},
		"clear": func(_ []string) error {
			Handler.clearTerminal()
			return nil
		},
		"clear_db": func(_ []string) error {
			return Handler.clearDatabase()
		},
		"order_history": func(_ []string) error {
			return Handler.orderHistory()
		},
		"accept_order":       Handler.acceptOrder,
		"return_to_courier":  Handler.returnToCourier,
		"process_customer":   Handler.processCustomer,
		"list_orders":        Handler.listOrders,
		"list_returns":       Handler.listReturns,
		"accept_orders_file": Handler.acceptOrdersFromFile,
	}
	return Handler
}

// Execute - Выполняет команду с переданными аргументами
func (h *Handler) Execute(command string, args []string) error {
	cmdFunc, exists := h.commands[command]
	if !exists {
		return fmt.Errorf("неизвестная команда - %s. Введите help для списка команд", command)
	}

	return cmdFunc(args)
}

// clearTerminal - Очищает экран терминала
func (h *Handler) clearTerminal() {
	fmt.Print("\033[H\033[2J")
}

// printHelp - Выводит справку по командам
func (h *Handler) printHelp() {
	fmt.Print(`Доступные команды:
	help                          - вывести список команд
	exit                          - завершить программу
	clear                         - очистить консоль

	accept_order <orderID> <clientID> <deadline> <weight> <cost> [package_type[+wrapper]]
		Принять заказ от курьера.
		deadline в формате "YYYY-MM-DDTHH:MM:SS",
		либо как относительная длительность (например, "30s" или "48h")
		weight - вес заказа в кг
		cost - стоимость заказа в рублях
		package_type - тип упаковки (box - коробка, bag - пакет, film - пленка)
		wrapper - дополнительная обертка (film - пленка)
		Примеры:
			accept_order 1 1 "48h" 5.0 100.0 box
			accept_order 1 1 "48h" 5.0 100.0 box+film
			accept_order 1 1 "2030-02-20T15:04:05" 5.0 100.0 bag+film

	return_to_courier <orderID>
		Вернуть заказ курьеру.

	process_customer <customerID> <action> <orderID1> [orderID2 ...]
		Выдать заказы или принять возврат клиента.
		action: "handout" или "return".

	list_orders <customerID> [pageSize <N>] [last <N>] [pvz]
		Получить список заказов с пагинацией скроллом.
		По умолчанию - размер страницы = 5

	list_returns pageSize <size>
		Получить список возвратов с интерактивной пагинацией (нажатие Enter – следующая страница).

	order_history
		Получить историю заказов.

	accept_orders_file <filename>
		Принять заказы от курьера из указанного JSON файла.

	clear_db
		Очистить базу данных.
`)
}

func (h *Handler) saveData() error {
	if h.service == nil || h.storage == nil {
		return errors.New("service or storage is nil")
	}
	data := h.service.Repo().GetAll()
	if err := h.storage.Save(data); err != nil {
		return fmt.Errorf("ошибка сохранения данных: %v", err)
	}
	return nil
}

// acceptOrder - Принимает заказ от курьера
func (h *Handler) acceptOrder(args []string) error {
	if err := validateAcceptOrderArgs(args); err != nil {
		return err
	}

	params, err := parseAcceptOrderParams(args)
	if err != nil {
		return err
	}

	if err = h.service.AcceptOrder(
		params.orderID,
		params.clientID,
		params.deadline,
		params.weight,
		params.cost,
		params.packageType,
		params.wrapper,
	); err != nil {
		return fmt.Errorf("ошибка при принятии заказа: %v", err)
	}

	if err = h.saveData(); err != nil {
		return err
	}

	fmt.Printf("Заказ принят. Итоговая стоимость: %.2f\n", h.service.Repo().GetAll()[params.orderID].Cost)
	return nil
}

// returnToCourier - Возвращает заказ курьеру
func (h *Handler) returnToCourier(args []string) error {
	if len(args) < 1 {
		return ErrInvalidReturnCourierArgs
	}

	orderID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("неверный формат orderID: %v", err)
	}

	if err = h.service.ReturnOrderToCourier(orderID); err != nil {
		return fmt.Errorf("ошибка при возврате заказа курьеру: %v", err)
	}

	err = h.saveData()
	if err != nil {
		return err
	}
	fmt.Println("Заказ возвращен курьеру:", orderID)

	return nil
}

// processCustomer - Обрабатывает выдачу или возврат заказов клиенту
func (h *Handler) processCustomer(args []string) error {
	if len(args) < 3 {
		return ErrInvalidProcessCustomerArgs
	}

	customerID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("неверный формат customerID: %w", err)
	}

	action := args[1]
	orderIDs := args[2:]
	now := time.Now()

	for _, idStr := range orderIDs {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return fmt.Errorf("неверный формат orderID: %w", err)
		}

		if err = h.processCustomerAction(action, id, customerID, now); err != nil {
			return err
		}
	}

	err = h.saveData()
	if err != nil {
		return err
	}

	return nil
}

func (h *Handler) processCustomerAction(action string, id, customerID int64, now time.Time) error {
	switch action {
	case "handout":
		if err := h.service.DeliverOrder(id, customerID, now); err != nil {
			return fmt.Errorf("ошибка при выдаче заказа: %v", err)
		}
		fmt.Printf("Заказ ID %d выдан клиенту %d\n", id, customerID)
	case "return":
		if err := h.service.ProcessReturnOrder(id, customerID, now); err != nil {
			return fmt.Errorf("ошибка при обработке возврата: %v", err)
		}
		fmt.Println("Возврат принят для заказа ", id)
	default:
		return fmt.Errorf("неизвестное действие: %s", action)
	}

	return nil
}

// orderHistory - Выводит историю заказов
func (h *Handler) orderHistory() error {
	orders := h.service.OrderHistory()
	if len(orders) == 0 {
		fmt.Println("База пуста")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if _, err := fmt.Fprintln(w, "ID\tКлиент\tСрок хранения\tСостояние\tВес\tСтоимость\tУпаковка\tОбновлен\tДоставлен\tВозврат"); err != nil {
		return fmt.Errorf("ошибка при записи заголовка: %v", err)
	}

	for _, order := range orders {
		delivery := "-"
		if order.DeliveredAt != nil {
			delivery = order.DeliveredAt.Format(timeLayout)
		}
		ret := "-"
		if order.ReturnedAt != nil {
			ret = order.ReturnedAt.Format(timeLayout)
		}

		if _, err := fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%.2f\t%.2f\t%s\t%s\t%s\t%s\n",
			order.ID,
			order.CustomerID,
			order.DeadlineAt.Format(timeLayout),
			order.State,
			order.Weight,
			order.Cost,
			formatPackageInfo(order),
			order.UpdatedAt.Format(timeLayout),
			delivery,
			ret); err != nil {
			return fmt.Errorf("ошибка при записи данных: %v", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("ошибка при выводе таблицы: %v", err)
	}

	return nil
}

// listReturns - Выводит список возвратов с пагинацией
func (h *Handler) listReturns(args []string) error {
	if len(args) < 2 || args[0] != "pageSize" {
		return ErrInvalidListReturnsArgs
	}

	pageSize, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("неверный формат размера страницы: %v", err)
	}
	if pageSize <= 0 {
		return ErrInvalidPageSize
	}

	returns := h.service.ListReturns()
	if len(returns) == 0 {
		fmt.Println("Нет данных для возвратов")
		return nil
	}

	return listReturnsPrintFull(returns, pageSize)
}

// listOrders - Выводит список заказов с пагинацией
func (h *Handler) listOrders(args []string) error {
	params, err := parseListOrdersParams(args)
	if err != nil {
		return err
	}

	ordersList := h.service.ListOrders(params.customerID, params.lastN, params.filterPVZ)
	if len(ordersList) == 0 {
		fmt.Println("Нет заказов")
		return nil
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("ошибка при настройке терминала: %v", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	terminal := term.NewTerminal(os.Stdin, "")

	currentPos := 0
	totalOrders := len(ordersList)

	displayFunc := func() error {
		return displayLOOrders(h, terminal, ordersList, currentPos, totalOrders, params.pageSize)
	}

	if err = displayFunc(); err != nil {
		return err
	}

	return handleLOKeyPress(displayFunc, &currentPos, totalOrders, params.pageSize)
}

// acceptOrdersFromFile - Принимает заказы из JSON файла
func (h *Handler) acceptOrdersFromFile(args []string) error {
	if len(args) < 1 {
		return ErrInvalidAcceptFileArgs
	}
	filename := args[0]
	if err := h.service.AcceptOrdersFromFile(filename); err != nil {
		return fmt.Errorf("ошибка при загрузке заказов из файла: %v", err)
	}

	err := h.saveData()
	if err != nil {
		return err
	}

	return nil
}

// clearDatabase - Очищает базу данных
func (h *Handler) clearDatabase() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Вы уверены, что хотите очистить базу? (Y/N): ")
	confirm, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("ошибка при чтении подтверждения: %v", err)
	}

	confirm = strings.TrimSpace(confirm)
	if strings.ToUpper(confirm) != "Y" {
		fmt.Println("Операция отменена.")
		return nil
	}

	h.service.Repo().SetAll(make(map[int64]model.Order))
	if err = h.storage.Save(h.service.Repo().GetAll()); err != nil {
		return fmt.Errorf("ошибка при очистке базы данных: %v", err)
	}
	fmt.Println("База успешно очищена.")

	return nil
}
