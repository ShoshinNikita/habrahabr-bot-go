package config


import (
	"io/ioutil"
	"encoding/json"
)

// ConfigurationData содержит конфигурационную информацию
type ConfigurationData struct {
	BotToken 		string 	`json:"botToken"` 		// token бота
	AppToken		string 	`json:"appToken"` 		// token приложения в advanced-log
	Delay			int64	`json:"delay"`			// в наносекундах
	SitePassword 	string 	`json:"password"`		// пароль от сайта
	AdvancedLogURL	string	`json:"advancedLogUrl"` // url of instant of advanced-log
}

// Data содержит конфигурационные данные
var Data ConfigurationData


// ReadConfig читает config.json и записывает данные в Data
func ReadConfig() error {
	raw, err := ioutil.ReadFile("data/config.json")
	if err != nil {
		return err
	}
	
	err = json.Unmarshal(raw, &Data)
	if err != nil {
		return err
	}

	return nil
}