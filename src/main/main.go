package main

import (
	"log"
	"os"
	"fmt"

	"bot"
	"logging"
	"website"
	"config"
)


func main() {
	// Чтение config.json (В первую очередь!)
	err := config.ReadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Инициализация advanced-log
	err = logging.Initialize()
	if err != nil {
		log.Fatal(err)
	}

	logging.LogEvent("Старт программы")
	

	// Инициализация бота
	logging.LogEvent("Инициализация бота")
	habrBot := bot.NewBot()
	
	args := os.Args[1:]
	if len(args) == 0 {

		logging.LogEvent("Запуск бота")
		go habrBot.StartPooling()

		logging.LogEvent("Запуск сайта")
		go website.RunSite(habrBot)

	} else if args[0] == "-bot" {

		logging.LogEvent("Запуск бота")
		go habrBot.StartPooling()

	} else if args[0] == "-web" {

		logging.LogEvent("Запуск сайта")
		go website.RunSite(habrBot)

	} else {
		fmt.Println("Неверные аргументы")
		logging.FatalErrorChan <- true
	}

	// Поток блокируется до появления фатальной ошибки
	// (при появлении фатальной ошибки программа завершится с кодом 1)
	<- logging.FatalErrorChan
}
