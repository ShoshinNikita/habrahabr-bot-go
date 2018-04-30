package bot

import (
	"sync"
	"os"
	"encoding/json"
	"io/ioutil"
	"time"

	"gopkg.in/telegram-bot-api.v4"

	artdb "articlesdb"
)

// remindersQueue хранит напоминания в виде очереди
type remindersQueue struct {
	Reminders  	[]Reminder
	lock		*sync.Mutex
	fileLock	*sync.Mutex
	path		string
}

// push добавляет напоминание в очередь
func (all *remindersQueue) push(r Reminder) {
	all.lock.Lock()
	defer all.lock.Unlock()
	all.Reminders = append(all.Reminders, r)
	all.updateRemindersFile()
}

// pop удаляет первое напоминание из очереди
func (all *remindersQueue) pop() {
	all.lock.Lock()
	defer all.lock.Unlock()
	all.Reminders = append([]Reminder{}, all.Reminders[1:]...)
	all.updateRemindersFile()
}

// updateRemindersFile обновляет файл
func (all *remindersQueue) updateRemindersFile() {
	all.fileLock.Lock()
	defer all.fileLock.Unlock()

	data, _ := json.Marshal(reminders.Reminders)
	ioutil.WriteFile(all.path, data, 0644)
}

var reminders remindersQueue



// readReminders читает из файла список напоминаний
func (bot *Bot) readReminders(path string) error {
	// Инициализация reminders
	reminders.lock = new(sync.Mutex)
	reminders.fileLock = new(sync.Mutex)
	reminders.path = path

	// Проверка, существует ли файл. Если нет, то создаётся
	if _, err := os.Stat(path); os.IsNotExist(err) {
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		// Запись пустого массива
		file.WriteString("[]")
	} else {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		
		err = json.Unmarshal(data, &reminders.Reminders)
		if err != nil {
			return err
		}
	}
	
	return nil
}


// checkOldReminders проверяет напоминания, которые хранились в файле
func (bot *Bot) checkOldReminders() {
	now := time.Now()
	for _, item := range reminders.Reminders {
		if item.Time < now.Unix() {
			go bot.remind(item.UserID, item.Text, time.Duration(0))
		} else {
			last := time.Unix(item.Time, 0).Sub(now)
			go bot.remind(item.UserID, item.Text, last)
		}
	}
}


// addToReminder добавляет напоминание
func (bot *Bot) addToReminder(callback *tgbotapi.CallbackQuery) {
	key := callback.Data[6:]
	text, err := artdb.Get(key)
	if err != nil {
		answer := tgbotapi.NewCallback(callback.ID, "Невозможно добавить статью в напоминания")
		bot.answerCallback(answer)
		return
	}
	id := callback.Message.Chat.ID
	
	reminders.push(Reminder{Text: text, Time: time.Now().Add(6 * time.Hour).Unix(), UserID: id})
	go bot.remind(id, text, 6 * time.Hour)
	
	bot.answerCallback(tgbotapi.NewCallback(callback.ID, "Напоминание создано"))
}


// remind отсылает напоминание пользователю
func (bot *Bot) remind(id int64, text string, t time.Duration) {
	timer := time.NewTimer(t)

	// Ждём
	<- timer.C
	
	reminders.pop()
	text = "#Напоминание\n" + text
	message := tgbotapi.NewMessage(id, text)
	message.ParseMode = "HTML"
	bot.send(message)
}