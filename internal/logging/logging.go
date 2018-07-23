package logging

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/ShoshinNikita/advanced-log-go"

	"github.com/ShoshinNikita/habrahabr-bot-go/internal/config"
)

// ErrorData содержит информацию об ошибке и о пользователе, вызвавшем ошибку
type ErrorData struct {
	Error    error
	Username string
	UserID   int64
	Command  string
	AddInfo  string // AdditionalInfo
}

// RequestData содержит информацию о запросе
type RequestData struct {
	Username string
	ID       int64
	Command  string
}

var remoteLog *advancedlog.LogAPI

// Initialize инициализирует remoteLog
func Initialize(debug bool) error {
	var err error
	remoteLog, err = advancedlog.NewLog(config.Data.AppToken, config.Data.AdvancedLogURL, debug)
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
	text := "User: " + data.Username + " ID: " + strconv.FormatInt(data.ID, 10)
	err := remoteLog.Log("request", data.Command, text, time.Now().Unix())
	if err != nil {
		log.Println(err)
	}
}

// LogError логгирует ошибку (программы)
func LogError(data ErrorData) {
	text := "User: " + data.Username + " Error: " + data.Error.Error()
	if data.AddInfo != "" {
		text += " AddInfo: " + data.AddInfo
	}

	err := remoteLog.Log("error", data.Command, text, time.Now().Unix())
	if err != nil {
		log.Println(err)
	}
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
	log.Println(funcName, message, err.Error())

	text := "FATAL ERROR Function: " + funcName + " Message: " + message + " Error: " + err.Error()
	err = remoteLog.Log("error", "", text, time.Now().Unix())
	if err != nil {
		log.Println(err)
	}
	os.Exit(1)
}
