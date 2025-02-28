package storage

import (
	"encoding/json"
	"io"
	"os"

	"gitlab.ozon.dev/gojhw1/pkg/model"
)

type OrderStorage interface {
	Save(map[int64]model.Order) error
	Load() (map[int64]model.Order, error)
}

type JSONStorage struct {
	FilePath string
}

// NewJSONStorage - создает новый экземпляр JSONStorage с указанным путем к файлу
func NewJSONStorage(filePath string) *JSONStorage {
	return &JSONStorage{FilePath: filePath}
}

// Save - сохраняет заказы в JSON файл
func (s *JSONStorage) Save(orders map[int64]model.Order) error {
	bytes, err := json.MarshalIndent(orders, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.FilePath, bytes, 0644)
}

// Load - загружает заказы из JSON файла
func (s *JSONStorage) Load() (map[int64]model.Order, error) {
	file, err := os.Open(s.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[int64]model.Order), nil
		}
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return make(map[int64]model.Order), nil
	}

	var ordersMap map[int64]model.Order
	err = json.Unmarshal(data, &ordersMap)
	if err != nil {
		return nil, err
	}

	return ordersMap, nil
}
