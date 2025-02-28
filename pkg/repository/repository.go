package repository

import (
	"errors"
	"fmt"

	"gitlab.ozon.dev/gojhw1/pkg/model"
)

var (
	ErrOrderAlreadyExists = errors.New("заказ уже существует")
	ErrOrderNotFound      = errors.New("заказ не существует")
	ErrInvalidOrderID     = errors.New("недопустимый ID заказа")
	ErrInvalidCustomerID  = errors.New("недопустимый ID клиента")
)

type Repository interface {
	Add(order model.Order) error
	Update(order model.Order) error
	Delete(id int64) error
	FindByID(id int64) (model.Order, error)
	List() []model.Order
	SetAll(orders map[int64]model.Order)
	GetAll() map[int64]model.Order
}

type InMemoryRepository struct {
	orders map[int64]model.Order
}

// NewInMemoryRepository - создает новый репозиторий заказов
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		orders: make(map[int64]model.Order),
	}
}

// Add - добавляет заказ в репозиторий
func (r *InMemoryRepository) Add(order model.Order) error {
	if order.ID <= 0 {
		return fmt.Errorf("%w: %d", ErrInvalidOrderID, order.ID)
	}

	if order.CustomerID <= 0 {
		return fmt.Errorf("%w: %d", ErrInvalidCustomerID, order.CustomerID)
	}

	if _, ok := r.orders[order.ID]; ok {
		return fmt.Errorf("%w: %d", ErrOrderAlreadyExists, order.ID)
	}
	r.orders[order.ID] = order

	return nil
}

// Update - обновляет уже существующий заказ
func (r *InMemoryRepository) Update(order model.Order) error {
	if _, ok := r.orders[order.ID]; !ok {
		return fmt.Errorf("%w: %d", ErrOrderNotFound, order.ID)
	}
	r.orders[order.ID] = order

	return nil
}

// Delete - удаляет заказ по ID
func (r *InMemoryRepository) Delete(id int64) error {
	if _, ok := r.orders[id]; !ok {
		return fmt.Errorf("%w: %d", ErrOrderNotFound, id)
	}
	delete(r.orders, id)

	return nil
}

// FindByID - находит заказ по ID
func (r *InMemoryRepository) FindByID(id int64) (model.Order, error) {
	order, ok := r.orders[id]
	if !ok {
		return model.Order{}, fmt.Errorf("%w: %d", ErrOrderNotFound, id)
	}

	return order, nil
}

// List - возвращает список всех заказов
func (r *InMemoryRepository) List() []model.Order {
	list := make([]model.Order, 0, len(r.orders))
	for _, order := range r.orders {
		list = append(list, order)
	}

	return list
}

// SetAll - устанавливает все заказы в репозиторий
func (r *InMemoryRepository) SetAll(orders map[int64]model.Order) {
	r.orders = make(map[int64]model.Order, len(orders))
	for k, v := range orders {
		r.orders[k] = v
	}
}

// GetAll - возвращает карту всех заказов
func (r *InMemoryRepository) GetAll() map[int64]model.Order {
	result := make(map[int64]model.Order, len(r.orders))
	for k, v := range r.orders {
		result[k] = v
	}
	return result
}
