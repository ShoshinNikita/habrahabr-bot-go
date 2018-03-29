package logging

import (
	"log"
	"time"
	"os"

	"gopkg.in/telegram-bot-api.v4"
	"bitbucket.org/weareprogrammers/advanced-log-go"
	"github.com/vjeantet/jodaTime"

	"config"
)


// ErrorData содержит информацию об ошибке и о пользователе, вызвавшем ошибку
type ErrorData struct {
	Error 		error
	Username 	string
	UserID 		int64
	Command 	string
	AddInfo		string // AdditionalInfo
}

// RequestData содержит информацию о запросе
type RequestData struct {
	Username 	string
	Command 	string
}

var remoteLog *advancedlog.LogAPI
// FatalErrorChan служит для блокировки основного потока
var FatalErrorChan chan bool


// Initialize инициализирует remoteLog
func Initialize() error {
	var err error

	FatalErrorChan = make(chan bool, 1)
	remoteLog, err = advancedlog.NewLog(config.Data.AppToken, config.Data.AdvancedLogURL)
	return err
}


// LogEvent логгирует события (например, рассылку статей)
func LogEvent(event string) {
	err := remoteLog.Log("event", "", event, time.Now().Unix())
	if err != nil {
		log.Println(err)
	}
}


// LogRequest логгирует запрос от пользователя
func LogRequest(data RequestData) {
	text := "User: " + data.Username
	err := remoteLog.Log("request", data.Command, text, time.Now().Unix())
	if err != nil {
		log.Println(err)
	}
}


// LogErrorAndNotify логгирует ошибку (программы) и отправляет пользователю информацию об ошибке (время)
func LogErrorAndNotify(data ErrorData, bot *tgbotapi.BotAPI) {
	text := "User: " + data.Username + " Error: " + data.Error.Error()
	if data.AddInfo != "" {
		text += "AddInfo: " + data.AddInfo
	}

	err := remoteLog.Log("error", data.Command, text, time.Now().Unix())
	if err != nil {
		log.Println(err)
	}

	// Отправление сообщения об ошибке
	text = "Что-то пошло не так. Время: " + GetCurrentTime() + "\nОб ошибках писать @Tirsias"
	message := tgbotapi.NewMessage(data.UserID, text)
	bot.Send(message)
}


// LogMinorError логгирует мелкие ошибки, которые произошли во время работы программы
func LogMinorError(funcName, message string, err error) {
	text := "Function: " + funcName + " Message: " + message + " Error: " + err.Error()
	err = remoteLog.Log("error", "", text, time.Now().Unix())
	if err != nil {
		log.Println(err)
	}
}


// LogFatalError логгирует фатальную ошибку, после чего завершает программу с кодом 1
func LogFatalError(funcName, message string, err error) {
	// На всякий случай
	log.Panicln(funcName, err.Error())

	text := "!!! FATAL !!! Function: " + funcName + " Message: " + message + " Error: " + err.Error()
	err = remoteLog.Log("error", "", text, time.Now().Unix())
	if err != nil {
		log.Println(err)
	}
	os.Exit(1)
}


// SendErrorToUser отправляет пользователю сообщение об ошибке (некорректный формат данных, отправленный пользователем)
func SendErrorToUser(text string, bot *tgbotapi.BotAPI, userID int64) {
	message := tgbotapi.NewMessage(userID, "Ошибка: " + text)
	bot.Send(message)
}


// GetCurrentTime возвращает текущее время
func GetCurrentTime() string {
	return jodaTime.Format("dd.MM.yyyy HH:mm:ss", time.Now())
}
