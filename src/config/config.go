package config


import (
	"errors"
	"flag"
)

// ConfigurationData содержит конфигурационную информацию
type ConfigurationData struct {
	BotToken 		string 		// token бота
	AppToken		string 		// token приложения в advanced-log
	Delay			int64		// в наносекундах
	SitePassword 	string		// пароль от сайта
	AdvancedLogURL	string		// url of instant of advanced-log
	PathToDataBase	string		// путь до базы данных
	PathToTimeFile	string		// путь до файла, в котором хранится время последних статей
	Debug			bool
}

// Data содержит конфигурационные данные
var Data ConfigurationData


// GetConfigInfo парсит флаги и заполняет Data
func GetConfigInfo() error {
	flag.StringVar(&Data.BotToken, "bToken", "", "token of a bot")
	flag.StringVar(&Data.AppToken, "aToken", "", "token of an app")
	flag.Int64Var(&Data.Delay, "delay", 600000000000, "delay of getting articles")
	flag.StringVar(&Data.SitePassword, "pass", "", "password for the site")
	flag.StringVar(&Data.AdvancedLogURL, "logUrl", "", "url of advanced-log")
	flag.StringVar(&Data.PathToDataBase, "dbPath", "", "path to the database")
	flag.StringVar(&Data.PathToTimeFile, "timePath", "", "path to the file 'lastArticleTime.json'")
	flag.BoolVar(&Data.Debug, "debug", false, "debug mode (default – false)")
	
	flag.Parse()

	if Data.BotToken == "" || Data.AppToken == "" || Data.AdvancedLogURL == "" || 
	Data.PathToDataBase == "" || Data.PathToTimeFile == "" {
		return errors.New("Miss the flag")
	}

	return nil
}