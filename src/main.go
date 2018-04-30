package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"	

	"articlesdb"
	"bot"
	"config"
	"userdb"
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

	// Инициализация базы данных c пользователями
	err = userdb.Open(config.Data.Prefix + "data/users.db")
	if err != nil {
		logging.LogFatalError("main", "попытка открыть базу данных с пользователями", err)
	}

	// Инициализация базы данных cо статьями
	err = articlesdb.Open(config.Data.Prefix + "data/articles.db")
	if err != nil {
		logging.LogFatalError("main", "попытка открыть базу данных со статьями", err)
	}

	// Инициализация бота
	logging.LogEvent("Инициализация бота")
	habrBot, err := bot.NewBot()
	if err != nil {
		logging.LogFatalError("main", "попытка залогиниться в бота", err)
	}

	// Запуск бота
	logging.LogEvent("Запуск бота")
	stopChan := make(chan struct{})
	go habrBot.StartPooling(stopChan)

	// Запуск сайта
	logging.LogEvent("Запуск сайта")
	go website.RunSite(habrBot)


	// Перехватываем сигналы
	sigChan := make(chan os.Signal, 1)
	// SIGTERM для Сервера (htop kill 15), SIGINT для Windows (Ctrl+C)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	// Ждём сигнала
	<- sigChan
	// Останавливаем бота
	close(stopChan)
	// Ждём пока все функции завершатся.
	// Из-за того, что бот больше не принимает новые сообщений, функции не будут вызываться
	time.Sleep(2 * time.Second)
	// Закрытие баз данных
	userdb.Close()
	articlesdb.Close()
	logging.LogEvent("Остановка работы")
	time.Sleep(500 * time.Millisecond)
}
