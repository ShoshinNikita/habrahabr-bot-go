package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
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

func LogInfo(format string, v ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	format = "[INFO] " + format

	log.Printf(format, v...)
}

// LogRequest логгирует запрос от пользователя
func LogRequest(data RequestData) {
	log.Printf("[REQ] User: %s ID: %d Cmd: %s\n", data.Username, data.ID, data.Command)
}

// LogError логгирует ошибку (программы)
func LogError(data ErrorData) {
	text := fmt.Sprintf("[ERR] User: %s ID: %d Cmd: %s Err: %s", data.Username, data.UserID,
		data.Command, data.Error)
	if data.AddInfo != "" {
		text += " AddInfo: " + data.AddInfo
	}

	log.Println(text)
}

// LogMinorError логгирует мелкие ошибки, которые произошли во время работы программы
func LogMinorError(funcName, message string, err error) {
	log.Printf("[ERR] Func: %s Err: %s AddInfo: %s\n", funcName, err, message)
}

// LogFatalError логгирует фатальную ошибку, после чего завершает программу с кодом 1
func LogFatalError(funcName, message string, err error) {
	log.Printf("[FATAL ERR] Func: %s Err: %s Msg: %s", funcName, err, message)
	os.Exit(1)
}
