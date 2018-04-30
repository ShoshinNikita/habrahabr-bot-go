package bot

import (
	"encoding/json"
	"io/ioutil" // чтение файлов
	"regexp"

	"gopkg.in/telegram-bot-api.v4" // Telegram api
	"github.com/jasonlvhit/gocron" // Job Scheduling Package

	"config"
	"userdb"      // взаимодействие с базой данных
	"logging" // логгирование
)


// Bot надстрройка над tgbotapi.BotAPI
type Bot struct {
	botAPI *tgbotapi.BotAPI
}


// NewBot инициализирует бота
func NewBot() (*Bot, error) {
	var err error

	// Инициализация бота
	var bot Bot
	bot.botAPI, err = tgbotapi.NewBotAPI(config.Data.BotToken)
	if err != nil {
		return nil, err
	}

	bot.botAPI.Buffer = 12 * 50

	// Чтение файла с напоминаниями
	err = bot.readReminders(config.Data.Prefix + "data/reminders.json")
	if err != nil {
		return nil, err
	}

	return &bot, nil
}


// StartPooling начинает перехватывать сообщения
func (bot *Bot) StartPooling(stopChan chan struct{}) {
	// Получение канала обновлений
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updateChannel, err := bot.botAPI.GetUpdatesChan(updateConfig)
	if err != nil {
		logging.LogFatalError("StartPooling", "попытка получить GetUpdatesChan", err)
	}

	// Чтение lastTime
	var lastTime LastArticlesTime
	raw, err := ioutil.ReadFile(config.Data.Prefix + "data/lastArticleTime.json")
	if err != nil {
		logging.LogFatalError("StartPooling", "попытка прочесть lastArticleTime.json", err)
	}
	json.Unmarshal(raw, &lastTime)


	// Страрт рассылки
	var commonScheduler, bestScheduler, clearArticles gocron.Scheduler
	commonScheduler.Every(config.Data.Delay).Seconds().Do(bot.mailout, &lastTime)
	// Запуск рассылки статей сразу
	commonScheduler.RunAll()
	// Рассылка запустится через 10 минут
	commonScheduler.Start()

	// Старт рассылки лучших статей каждый день в 21:00
	bestScheduler.Every(1).Day().At("21:00").Do(bot.mailoutBestArticles)
	bestScheduler.Start()

	// Очистка списка статей каждые 7 дней
	clearArticles.Every(7).Days().Do(clearArticlesDB)
	clearArticles.Start()

	// Проверка старых напоминаний
	go bot.checkOldReminders()

	// Long pooling
	for update := range updateChannel {
		select {
			case <-stopChan: {
				// При закрытии канала завершаем выполнение функции.
				// В этот цикл мы заходим только тогда, когда кто-то написал боту.
				// Если кто-то писал, то функция сразу завершается, если нет – завершается вместе с программой
				// При этом может потеряться часть запросов (они будут необработанны), но с этим ничего нельзя сделать
				logging.LogEvent("Остановка Long Pooling (происходит не всегда)")
				return
			}
			default: {
				go bot.distributeUpdate(update)
			}
		}
	}
}

// distributeUpdate обрабатывает новые сообщения
func (bot *Bot) distributeUpdate(update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		if !bot.distributeCallback(update.CallbackQuery) {
			answer := tgbotapi.NewCallback(update.CallbackQuery.ID, "Неверный callback")
			bot.answerCallback(answer)
		}
	} else if update.Message != nil {
		if !bot.distributeMessages(update.Message) {
			message := tgbotapi.NewMessage(update.Message.Chat.ID, "Неверная команда. Для справки введите /help")
			message.ReplyToMessageID = update.Message.MessageID
			bot.send(message)
		}
	}
}


// distributeMessages распределяет сообщения по goroutine'ам
// Если сообщение не получилось распределить, то возвращается false, иначе – true
func (bot *Bot) distributeMessages(message *tgbotapi.Message) bool {
	var isRightCommand = false
	var site string

	command := message.Command()
	if command == "" {
		if res, _ := regexp.MatchString(habrArticleRegexPattern, message.Text); res {
			go bot.sendIV(userCommand{message, habr})
			isRightCommand = true
		} else if res, _ = regexp.MatchString(geekArticleRegexPattern, message.Text); res {
			go bot.sendIV(userCommand{message, geek})
			isRightCommand = true
		}

		// Логгируется, только если сообщение похоже на ссылку
		if isRightCommand {
			logging.LogRequest(logging.RequestData{Command: "InstantView", Username: message.Chat.UserName, ID: message.Chat.ID})
		}
	} else {
		// Логгирование запроса
		logging.LogRequest(logging.RequestData{Command: "/" + command, Username: message.Chat.UserName, ID: message.Chat.ID})

		// Рассматривается отдельно, т.к. команды используется без префиксов
		if command == "help" {
			go bot.help(message)
			return true
		} else if command == "start" {
			go bot.start(message)
			return true
		} else if command == "show_keyboard" {
			go bot.showKeyboard(message)
			return true
		} else if command == "hide_keyboard" {
			go bot.hideKeyboard(message)
			return true
		}

		// Длина всегда > 5
		if len(command) <= 5 {
			return false
		}
		if prefix := command[:5]; prefix == "geek_" {
			site = geek
		} else if prefix == "habr_" {
			site = habr
		} else {
			return false
		}
		command = command[5:]

		switch command {
			// start mailout
			case "start": {
				go bot.startMailout(userCommand{message, site})
				isRightCommand = true
			}
			case "stop": {
				go bot.stopMailout(userCommand{message, site})
				isRightCommand = true
			}
			case "tags": {
				go bot.getStatus(userCommand{message, site})
				isRightCommand = true
			}
			case "add_tags": {
				go bot.addTags(userCommand{message, site})
				isRightCommand = true
			}
			case "del_tags": {
				go bot.delTags(userCommand{message, site})
				isRightCommand = true
			}
			case "del_all_tags": {
				go bot.delAllTags(userCommand{message, site})
				isRightCommand = true
			}
			case "best": {
				go bot.getBest(userCommand{message, site})
				isRightCommand = true
			}
			case "copy_tags": {
				go bot.copyTags(userCommand{message, site})
				isRightCommand = true
			}
		}
	}

	return isRightCommand
}


// distributeCallback распределяет callback'и по goroutine'ам
// Если callback не получилось распределить, то возвращается false, иначе – true
func (bot *Bot) distributeCallback(callback *tgbotapi.CallbackQuery) bool {
	logging.LogRequest(logging.RequestData{Command: callback.Data, ID: int64(callback.From.ID), Username: callback.From.UserName})
	
	rightCallback := false
	if len(callback.Data) > 6 && callback.Data[:6] == "remind" {
		rightCallback = true
		go bot.addToReminder(callback)
	}

	return rightCallback
}


// send отправляет сообщение
func (bot *Bot) send(msg tgbotapi.MessageConfig) {
	bot.botAPI.Send(msg)
}


// answerCallback отвечает на callback
func (bot *Bot) answerCallback(answer tgbotapi.CallbackConfig) {
	bot.botAPI.AnswerCallbackQuery(answer)
}


// Notify отправляет пользователям сообщение, полученное через сайт
func (bot *Bot) Notify(sMessage string) error {
	users, err := userdb.GetAllUsers()
	if err != nil {
		logging.LogMinorError("Notify", "попытка получить список пользователей", err)
		return err
	}

	for _, user := range users {
		message := tgbotapi.NewMessage(user.ID, sMessage)
		message.ParseMode = "HTML"
		bot.send(message)
	}

	return nil
}