package app

import (
	"errors"
	"fmt"
	"log"

	"github.com/chzyer/readline"
	"gitlab.ozon.dev/gojhw1/pkg/handler/commands"
	"gitlab.ozon.dev/gojhw1/pkg/handler/input"
)

type App struct {
	inputHandler *input.Handler
	cmdHandler   *commands.Handler
}

func New(inputHandler *input.Handler, cmdHandler *commands.Handler) *App {
	return &App{
		inputHandler: inputHandler,
		cmdHandler:   cmdHandler,
	}
}

func (a *App) Close() {
	if a.inputHandler != nil {
		a.inputHandler.Close()
	}
}

func (a *App) StartAndWatch() error {
	printWelcome()

	for {
		line, err := a.inputHandler.ReadLine()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) {
				fmt.Println("Выход из программы")
				a.Close()
				return nil
			}
			return err
		}

		command, args := a.inputHandler.ProcessLine(line)
		if command == "" {
			continue
		}

		if err = a.cmdHandler.Execute(command, args); err != nil {
			log.Printf("ошибка команды %s: %v\n", command, err)
		}
	}
}

func printWelcome() {
	fmt.Println(`
        ____              __      ___   ___________    
       / __ \____  __  __/ /____ |__ \ / ____/ ___/    
      / /_/ / __ \/ / / / __/ _ \__/ //___ \/ __ \     
     / _, _/ /_/ / /_/ / /_/  __/ __/____/ / /_/ /     
    /_/ |_|\____/\__,_/\__/\___/____/_____/\____/      
                                                       
    `)
	fmt.Println("Добро пожаловать в менеджер ПВЗ. Введите help для списка команд.")
}
