package config

import (
	"errors"
	"flag"
)

// ConfigurationData содержит конфигурационную информацию
type ConfigurationData struct {
	BotToken string // token бота
	Delay    uint64 // в секундах
	Rate     uint64 // в милисекундах
}

// Data содержит конфигурационные данные
var Data ConfigurationData

// GetConfigInfo парсит флаги и заполняет Data
func GetConfigInfo() error {
	var nanoseconds uint64
	flag.StringVar(&Data.BotToken, "bToken", "", "token of a bot")
	flag.Uint64Var(&nanoseconds, "delay", 1200000000000, "delay of getting articles (nanoseconds)")
	flag.Uint64Var(&Data.Rate, "rate", 500, "delay between sending of messages (milliseconds)")

	flag.Parse()

	// Получаем задержку в секундах
	Data.Delay = nanoseconds / 1e9

	if Data.BotToken == "" {
		return errors.New("botToken is missed")
	}

	return nil
}
