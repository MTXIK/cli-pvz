package main

import (
	"log"

	"gitlab.ozon.dev/gojhw1/pkg/app"
	"gitlab.ozon.dev/gojhw1/pkg/handler/commands"
	"gitlab.ozon.dev/gojhw1/pkg/handler/input"
	"gitlab.ozon.dev/gojhw1/pkg/repository"
	"gitlab.ozon.dev/gojhw1/pkg/service"
	"gitlab.ozon.dev/gojhw1/pkg/storage"
)

const storageFile = "./data/storage.json"

func main() {
	repo := repository.NewInMemoryRepository()
	jsonStorage := storage.NewJSONStorage(storageFile)

	data, err := jsonStorage.Load()
	if err != nil {
		log.Fatalf("ошибка загрузки данных: %v", err)
	}
	repo.SetAll(data)

	orderService := service.NewOrderService(repo)
	cmdHandler := commands.NewHandler(orderService, jsonStorage)

	inputHandler, err := input.NewHandler()
	if err != nil {
		log.Fatalf("ошибка инициализации readline: %v", err)
	}

	application := app.New(inputHandler, cmdHandler)

	if err = application.StartAndWatch(); err != nil {
		application.Close()
		log.Fatalf("ошибка работы приложения: %v", err)
	}
}
