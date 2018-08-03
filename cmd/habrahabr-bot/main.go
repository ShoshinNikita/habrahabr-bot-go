package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ShoshinNikita/habrahabr-bot-go/internal/bot"
	"github.com/ShoshinNikita/habrahabr-bot-go/internal/config"
	"github.com/ShoshinNikita/habrahabr-bot-go/internal/logging"
	"github.com/ShoshinNikita/habrahabr-bot-go/internal/userdb"
)

func main() {
	// Получение конфигурационной информации
	err := config.GetConfigInfo()
	if err != nil {
		log.Fatal(err)
	}

	// Инициализация advanced-log
	logging.LogInfo("Старт программы")

	// Инициализация базы данных c пользователями
	err = userdb.Open("data/users.db")
	if err != nil {
		logging.LogFatalError("main", "попытка открыть базу данных с пользователями", err)
	}

	// Получение корректных id
	err = bot.ParseCorrectIDS("data/ids.json")
	if err != nil {
		logging.LogFatalError("main", "попытка распарсить список id", err)
	}

	// Инициализация бота
	logging.LogInfo("Инициализация бота")
	habrBot, err := bot.NewBot()
	if err != nil {
		logging.LogFatalError("main", "попытка залогиниться в бота", err)
	}

	// Запуск бота
	logging.LogInfo("Запуск бота")
	stopChan := make(chan struct{})
	go habrBot.StartPooling(stopChan)

	// Перехватываем сигналы
	sigChan := make(chan os.Signal, 1)
	// SIGTERM для Сервера (htop kill 15), SIGINT для Windows (Ctrl+C)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	// Ждём сигнала
	<-sigChan
	// Останавливаем бота
	close(stopChan)
	// Ждём пока все функции завершатся.
	// Из-за того, что бот больше не принимает новые сообщений, функции не будут вызываться
	time.Sleep(2 * time.Second)
	// Закрытие базы данных
	userdb.Close()
	logging.LogInfo("Остановка работы")
	time.Sleep(500 * time.Millisecond)
}
