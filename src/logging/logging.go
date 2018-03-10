package logging

import (
	"os"
	"sync"
	"time"

	"gopkg.in/telegram-bot-api.v4"
	"github.com/vjeantet/jodaTime"
)

// ErrorData содержит информацию об ошибке и о пользователе, вызвавшем ошибку
type ErrorData struct {
	Err error
	Username string
	UserID int64
}

var (
	// FatalErrorChan – канал, сообщения от которого ждёт основная программа
	FatalErrorChan chan bool // Здесь происходит объявление канала. Его инициализация – в OpenLogFiles

	sendingErrorLog *os.File // файл для логгирования ошибок, возникших при отправлении сообщения
	sendingMutex sync.Mutex

	eventsLog *os.File // файл для логгирования событий
	eventsMutex sync.Mutex

	programErrorLog *os.File // файл для логгирования ошибок программы
	programMutex sync.Mutex

	minorErrorLog *os.File
	minorMutex sync.Mutex

	fatalErrorLog *os.File
)


// GetCurrentTime возвращает текущее время
func GetCurrentTime() string {
	return jodaTime.Format("dd.MM.yyyy HH:mm:ss", time.Now())
}


// OpenLogFiles открывает файлы для ведения логов.
func OpenLogFiles(path string) (error){
	// Инициализация канала
	FatalErrorChan = make(chan bool, 1)

	var err error
	sendingErrorLog, err = os.OpenFile(path + "/sendingError.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	eventsLog, err = os.OpenFile(path + "/events.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	programErrorLog, err = os.OpenFile(path + "/programError.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		return err
	}	
	minorErrorLog, err = os.OpenFile(path + "/minorError.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	fatalErrorLog, err = os.OpenFile(path + "/fatalError.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	return nil
}


// LogSendingError логгирует ошибки, возникшие при отправке сообщений
func LogSendingError(data ErrorData) {
	sendingMutex.Lock()
	defer sendingMutex.Unlock()
	
	text := GetCurrentTime() + "\tFrom " + data.Username + "\tError: " + data.Err.Error() + "\n"
	sendingErrorLog.WriteString(text)
}


// LogEvent логгирует события (например, рассылку статей)
func LogEvent(event string) {
	eventsMutex.Lock()
	defer eventsMutex.Unlock()

	text := GetCurrentTime() + "\t" + event + "\n"
	eventsLog.WriteString(text)
}


// LogErrorAndNotify логгирует ошибку (программы) и отправляет пользователю информацию об ошибке (время)
func LogErrorAndNotify(data ErrorData, bot *tgbotapi.BotAPI) {
	programMutex.Lock()
	defer programMutex.Unlock()
	
	sTime := GetCurrentTime()
	text := sTime + "\tFrom " + data.Username + "\tError: " + data.Err.Error() + "\n"
	programErrorLog.WriteString(text)

	// Отправление сообщения об ошибке
	text = "Что-то пошло не так. Время: " + sTime + "\nОб ошибках писать @Tirsias"
	message := tgbotapi.NewMessage(data.UserID, text)
	bot.Send(message)
}


// LogMinorError логгирует мелкие ошибки, которые произошли во время работы программы
func LogMinorError(funcName string, err error) {
	minorMutex.Lock()
	defer minorMutex.Unlock()

	text := GetCurrentTime() + "\t Error in function '" + funcName + "': " + err.Error() + "\n"
	minorErrorLog.WriteString(text)
}


// LogFatalError логгирует фатальную ошибку, после чего завершает программу с кодом 1
func LogFatalError(funcName string, err error) {
	text := GetCurrentTime() + "\t Error in function '" + funcName + "': " + err.Error() + "\n"
	fatalErrorLog.WriteString(text)
	os.Exit(1)
}


// SendErrorToUser отправляет пользователю сообщение об ошибке (некорректный формат данных, отправленный пользователем)
func SendErrorToUser(text string, bot *tgbotapi.BotAPI, userID int64) {
	message := tgbotapi.NewMessage(userID, "Ошибка: " + text)
	bot.Send(message)
}
