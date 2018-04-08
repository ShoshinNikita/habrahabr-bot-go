package main

import (
	"log"

	"bot"
	"config"
	"db"
	"logging"
	"website"
)


func main() {
	// Получение конфигурационной информации
	err := config.GetConfigInfo()
	if err != nil {
		log.Fatal(err)
	}

	// Инициализация advanced-log
	err = logging.Initialize(config.Data.Debug)
	if err != nil {
		log.Fatal(err)
	}
	logging.LogEvent("Старт программы")

	// Инициализация базы данных
	db.Open(config.Data.PathToDataBase)

	// Инициализация бота
	logging.LogEvent("Инициализация бота")
	habrBot := bot.NewBot()

	// Запуск бота
	logging.LogEvent("Запуск бота")
	go habrBot.StartPooling()

	// Запуск сайта
	logging.LogEvent("Запуск сайта")
	go website.RunSite(habrBot)

	// Поток блокируется до появления фатальной ошибки
	// (при появлении фатальной ошибки программа завершится с кодом 1)
	<- logging.FatalErrorChan
}
