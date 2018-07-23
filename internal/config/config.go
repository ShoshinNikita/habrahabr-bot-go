package config

import (
	"errors"
	"flag"
)

// ConfigurationData содержит конфигурационную информацию
type ConfigurationData struct {
	BotToken       string // token бота
	AppToken       string // token приложения в advanced-log
	Delay          uint64 // в секундах
	Rate           uint64 // в милисекундах
	SitePassword   string // пароль от сайта
	AdvancedLogURL string // url of instant of advanced-log
	Port           string // порт, на котором будет запускаться сайт
	Debug          bool
}

// Data содержит конфигурационные данные
var Data ConfigurationData

// GetConfigInfo парсит флаги и заполняет Data
func GetConfigInfo() error {
	var nanoseconds uint64
	flag.StringVar(&Data.BotToken, "bToken", "", "token of a bot")
	flag.StringVar(&Data.AppToken, "aToken", "", "token of an app")
	flag.Uint64Var(&nanoseconds, "delay", 1200000000000, "delay of getting articles (nanoseconds)")
	flag.Uint64Var(&Data.Rate, "rate", 500, "delay between sending of messages (milliseconds)")
	flag.StringVar(&Data.SitePassword, "pass", "", "password for the site")
	flag.StringVar(&Data.AdvancedLogURL, "logUrl", "", "url of advanced-log")
	flag.StringVar(&Data.Port, "port", "8080", "port for website, without ':'")
	flag.BoolVar(&Data.Debug, "debug", false, "debug mode (default – false)")

	flag.Parse()

	// Получаем задержку в секундах
	Data.Delay = nanoseconds / 1e9

	if Data.BotToken == "" {
		return errors.New("botToken is missed")
	}

	return nil
}
