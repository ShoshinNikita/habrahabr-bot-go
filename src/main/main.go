package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"fmt"

	"bot"
	"logging"
	"website"
)


func main() {
	// В первую очередь! Открываются файлы для логов
	err := logging.OpenLogFiles("logs")
	if err != nil {
		log.Fatal(err)
	}

	// Чтение config.json
	var config bot.ConfigData
	logging.LogEvent("Чтение config.json")
	raw, err := ioutil.ReadFile("./data/config.json")
	if err != nil {
		logging.LogFatalError("main", err)
	}
	err = json.Unmarshal(raw, &config)
	if err != nil {
		logging.LogFatalError("main", err)
	}

	// Инициализация бота
	logging.LogEvent("Инициализация бота")
	habrBot := bot.NewBot(config)
	
	args := os.Args[1:]
	if len(args) == 0 {

		logging.LogEvent("Запуск бота")
		go habrBot.StartPooling()

		logging.LogEvent("Запуск сайта")
		go website.RunSite(habrBot, config.SitePassword)

	} else if args[0] == "-bot" {

		logging.LogEvent("Запуск бота")
		go habrBot.StartPooling()

	} else if args[0] == "-web" {

		logging.LogEvent("Запуск сайта")
		go website.RunSite(habrBot, config.SitePassword)

	} else {
		fmt.Println("Неверные аргументы")
	}

	// Поток блокируется до появления фатальной ошибки
	// (канал будет пустым всегда, т.к. при появлении фатальной ошибки программа завершится с кодом 1)
	<- logging.FatalErrorChan
}
