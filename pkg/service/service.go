package service

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"gitlab.ozon.dev/gojhw1/pkg/model"
	"gitlab.ozon.dev/gojhw1/pkg/repository"
)

var (
	ErrStorageDeadlinePassed = errors.New("срок хранения в прошлом")
	ErrOrderExists           = errors.New("заказ уже существует")
	ErrDeadlineNotExpired    = errors.New("срок хранения заказа еще не истек")
	ErrOrderAlreadyDelivered = errors.New("заказ уже доставлен клиенту, возврат невозможен")
	ErrWrongCustomer         = errors.New("заказ принадлежит другому клиенту")
	ErrWrongState            = errors.New("заказ нельзя выдать – неверное состояние")
	ErrStorageExpired        = errors.New("срок хранения заказа истек")
	ErrNotDelivered          = errors.New("заказ не был выдан, возврат невозможен")
	ErrReturnExpired         = errors.New("срок возврата заказа истек")
	ErrOpenFile              = errors.New("ошибка при открытии файла принятия заказов")
	ErrReadFile              = errors.New("ошибка при чтении файла принятия заказов")
	ErrParseFile             = errors.New("ошибка при разборе файла принятия заказов")
	ErrInvalidDateFormat     = errors.New("неверный формат даты или длительности")
	ErrNegativeWeight        = errors.New("вес должен быть положительным числом")
	ErrNegativeCost          = errors.New("стоимость должна быть положительным числом")
)

const ReturnedAt = 48 * time.Hour
const timeLayout = "2006-01-02T15:04:05"

// OrderService - структура сервиса для работы с заказами
type OrderService struct {
	repo repository.Repository
}

// NewOrderService - создаёт новый сервис с переданным репозиторием
func NewOrderService(repo repository.Repository) *OrderService {
	return &OrderService{
		repo: repo,
	}
}

// Repo - возвращает репозиторий, связанный с сервисом
func (s *OrderService) Repo() repository.Repository {
	return s.repo
}

// AcceptOrder - принимает заказ, если он корректен и не просрочен
func (s *OrderService) AcceptOrder(id, customerID int64, deadline time.Time, weight, cost float64, packageType *model.PackageType, wrapper *model.WrapperType) error {
	now := time.Now()
	if now.After(deadline) {
		return fmt.Errorf("%w: %v \n Текущая дата: %v", ErrStorageDeadlinePassed, deadline, now)
	}
	if order, _ := s.repo.FindByID(id); order.ID == id {
		return fmt.Errorf("%w: Id %d", ErrOrderExists, id)
	}
	if weight <= 0 {
		return fmt.Errorf("%w: %v", ErrNegativeWeight, weight)
	}
	if cost <= 0 {
		return fmt.Errorf("%w: %v", ErrNegativeCost, cost)
	}

	finalCost := cost

	if packageType != nil {
		factory := newPackagerFactory()
		packager, err := factory.createPackager(packageType, wrapper)
		if err != nil {
			return fmt.Errorf("ошибка создания упаковщика: %w", err)
		}

		if err = packager.validateWeight(weight); err != nil {
			return fmt.Errorf("ошибка проверки веса для упаковки %s: %w", *packageType, err)
		}

		finalCost += packager.getAdditionalCost()
	}

	order := model.Order{
		ID:          id,
		CustomerID:  customerID,
		DeadlineAt:  deadline,
		State:       model.StateAccepted,
		UpdatedAt:   now,
		Weight:      weight,
		Cost:        finalCost,
		PackageType: packageType,
		Wrapper:     wrapper,
	}

	return s.repo.Add(order)
}

// ReturnOrderToCourier - возвращает заказ курьеру, если условия возврата соблюдены
func (s *OrderService) ReturnOrderToCourier(id int64) error {
	now := time.Now()
	order, err := s.repo.FindByID(id)
	if err != nil {
		return fmt.Errorf("ошибка при возврата заказа курьеру Id %d: %w", id, err)
	}
	if now.Before(order.DeadlineAt) && order.State != model.StateReturned {
		return fmt.Errorf("%w: %v\n текущая дата: %v", ErrDeadlineNotExpired, order.DeadlineAt, now)
	}
	if order.State == model.StateDelivered {
		return fmt.Errorf("%w: ID %d", ErrOrderAlreadyDelivered, id)
	}

	return s.repo.Delete(id)
}

// DeliverOrder - доставляет заказ клиенту, если заказ принадлежит клиенту и не просрочен
func (s *OrderService) DeliverOrder(id, customerID int64, now time.Time) error {
	order, err := s.repo.FindByID(id)
	if err != nil {
		return fmt.Errorf("ошибка при доставке заказа Id %d: %w", id, err)
	}
	if order.CustomerID != customerID {
		return fmt.Errorf("%w: ID %d", ErrWrongCustomer, id)
	}
	if order.State != model.StateAccepted {
		return fmt.Errorf("%w: ID %d", ErrWrongState, id)
	}
	if now.After(order.DeadlineAt) {
		return fmt.Errorf("%w: %v \n Текущая дата: %v", ErrStorageExpired, order.DeadlineAt, now)
	}

	order.State = model.StateDelivered
	order.UpdatedAt = now
	order.DeliveredAt = &now

	return s.repo.Update(order)
}

// ProcessReturnOrder - обрабатывает возврат заказа от клиента, если соблюдены условия возврата
func (s *OrderService) ProcessReturnOrder(id, customerID int64, now time.Time) error {
	order, err := s.repo.FindByID(id)
	if err != nil {
		return fmt.Errorf("ошибка при возврате заказа Id %d: %w", id, err)
	}
	if order.CustomerID != customerID {
		return fmt.Errorf("%w: ID %d", ErrWrongCustomer, id)
	}
	if order.State != model.StateDelivered {
		return fmt.Errorf("%w: ID %d", ErrNotDelivered, id)
	}
	if now.Sub(*order.DeliveredAt) > ReturnedAt {
		return fmt.Errorf("%w: %v \n Текущая дата: %v", ErrReturnExpired, order.DeliveredAt, now)
	}

	order.State = model.StateReturned
	order.UpdatedAt = now
	order.ReturnedAt = &now

	return s.repo.Update(order)
}

// OrderHistory - возвращает историю заказов, отсортированную по времени обновления (от новых к старым)
func (s *OrderService) OrderHistory() []model.Order {
	history := s.repo.List()
	sort.Slice(history, func(i, j int) bool {
		return history[i].UpdatedAt.After(history[j].UpdatedAt)
	})

	return history
}

// ListReturns - возвращает список возвращенных заказов для указанной страницы и размера страницы
func (s *OrderService) ListReturns() []model.Order {
	var returnsList []model.Order
	for _, order := range s.repo.List() {
		if order.State == model.StateReturned {
			returnsList = append(returnsList, order)
		}
	}

	sort.Slice(returnsList, func(i, j int) bool {
		return returnsList[i].ReturnedAt.After(*returnsList[j].ReturnedAt)
	})

	return returnsList
}

// ListOrders - возвращает список заказов клиента с возможностью фильтрации и ограничения количества
func (s *OrderService) ListOrders(customerID int64, lastN int, filterPVZ bool) []model.Order {
	var ordersList []model.Order
	for _, order := range s.repo.List() {
		if order.CustomerID != customerID {
			continue
		}
		if filterPVZ && (order.State != model.StateAccepted || time.Now().After(order.DeadlineAt)) {
			continue
		}
		ordersList = append(ordersList, order)
	}
	sort.Slice(ordersList, func(i, j int) bool {
		return ordersList[i].UpdatedAt.After(ordersList[j].UpdatedAt)
	})
	if lastN > 0 && len(ordersList) > lastN {
		ordersList = ordersList[:lastN]
	}

	return ordersList
}

// AcceptOrdersFromFile - принимает заказы из файла с форматом JSON
func (s *OrderService) AcceptOrdersFromFile(filename string) error {
	orders, err := readOrdersFromFile(filename)
	if err != nil {
		return err
	}

	for _, order := range orders {
		deadline, err := parseDeadline(order.DeadlineAt)
		if err != nil {
			return err
		}

		packageType, wrapper := processPackaging(order.PackageType, order.Wrapper)

		if err = s.AcceptOrder(
			order.ID,
			order.CustomerID,
			deadline,
			order.Weight,
			order.Cost,
			packageType,
			wrapper,
		); err != nil {
			return fmt.Errorf("ошибка при принятии заказа %d: %w", order.ID, err)
		}
		fmt.Printf("заказ %d принят\n", order.ID)
	}

	return nil
}
