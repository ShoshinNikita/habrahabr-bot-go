package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil" // чтение файлов
	"regexp"
	"time"

	"github.com/jasonlvhit/gocron" // Job Scheduling Package
	"gopkg.in/telegram-bot-api.v4" // Telegram api

	"github.com/ShoshinNikita/habrahabr-bot-go/internal/config"
	"github.com/ShoshinNikita/habrahabr-bot-go/internal/logging" // логгирование
)

// Bot надстрройка над tgbotapi.BotAPI
type Bot struct {
	botAPI   *tgbotapi.BotAPI
	messages chan tgbotapi.MessageConfig
	articles chan article
}

// Список id, с которыми бот может взаимодействовать
var correctIDs = []int64{}

// ParseCorrectIDS парсит json-файл, в котором содержится список корректных id
func ParseCorrectIDS(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &correctIDs)

	msg := fmt.Sprintf("Корректные id: %v", correctIDs)
	logging.LogEvent(msg)

	return err
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
	bot.messages = make(chan tgbotapi.MessageConfig, 300)
	bot.articles = make(chan article, 60)

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

	allArticles := struct {
		HabrArticles []string `json:"habr"`
	}{[]string{}}

	// Чтение lastArticles.json
	raw, err := ioutil.ReadFile("data/lastArticles.json")
	if err != nil {
		logging.LogFatalError("StartPooling", "попытка прочесть lastArticles.json", err)
	}
	json.Unmarshal(raw, &allArticles)

	// Инициализация oldArticles
	oldArticles = newSmartQueue(60, allArticles.HabrArticles)

	// Старт рассылки
	go bot.mailout()

	// Старт рассылки лучших статей каждый день в 21:00
	gocron.Every(1).Day().At("21:00").Do(bot.mailoutBestArticles)
	gocron.Start()

	go bot.sendWrapper(config.Data.Rate)

	// Long pooling
	for update := range updateChannel {
		select {
		case <-stopChan:
			{
				// При закрытии канала завершаем выполнение функции.
				// В этот цикл мы заходим только тогда, когда кто-то написал боту.
				// Если кто-то писал, то функция сразу завершается, если нет – завершается вместе с программой
				// При этом может потеряться часть запросов (они будут необработанны), но с этим ничего нельзя сделать
				logging.LogEvent("Остановка Long Pooling")
				return
			}
		default:
			{
				go bot.distributeUpdate(update)
			}
		}
	}
}

// distributeUpdate обрабатывает новые сообщения
func (bot *Bot) distributeUpdate(update tgbotapi.Update) {
	var isCorrectID = func(id int64) bool {
		for i := range correctIDs {
			if correctIDs[i] == id {
				return true
			}
		}
		return false
	}

	if update.Message != nil {
		if !isCorrectID(update.Message.Chat.ID) {
			text := fmt.Sprintf("Wrong ID: %d Username: %s Text: %s", update.Message.Chat.ID,
				update.Message.Chat.UserName, update.Message.Text)
			logging.LogEvent(text)

			message := tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный ID. Для подробностей писать @ShoshinNikita")
			bot.messages <- message
			return
		}

		if !bot.distributeMessages(update.Message) {
			message := tgbotapi.NewMessage(update.Message.Chat.ID, "Неверная команда. Для справки введите /help")
			message.ReplyToMessageID = update.Message.MessageID
			bot.messages <- message
		}
	}
}

// distributeMessages распределяет сообщения по goroutine'ам
// Если сообщение не получилось распределить, то возвращается false, иначе – true
func (bot *Bot) distributeMessages(message *tgbotapi.Message) bool {
	var isRightCommand = true

	command := message.Command()
	if command == "" {
		if res, _ := regexp.MatchString(habrArticleRegexPattern, message.Text); res {
			go bot.sendIV(message)
			logging.LogRequest(logging.RequestData{Command: "InstantView", Username: message.Chat.UserName, ID: message.Chat.ID})
		}
	} else {
		// Логгирование запроса
		logging.LogRequest(logging.RequestData{Command: "/" + command, Username: message.Chat.UserName, ID: message.Chat.ID})

		switch command {
		case "help":
			{
				go bot.help(message)
			}
		case "start":
			{
				go bot.start(message)
			}
		case "stop":
			{
				go bot.stopMailout(message)
				isRightCommand = true
			}
		case "tags":
			{
				go bot.getStatus(message)
				isRightCommand = true
			}
		case "add_tags":
			{
				go bot.addTags(message)
				isRightCommand = true
			}
		case "del_tags":
			{
				go bot.delTags(message)
				isRightCommand = true
			}
		case "del_all_tags":
			{
				go bot.delAllTags(message)
				isRightCommand = true
			}
		case "best":
			{
				go bot.getBest(message)
				isRightCommand = true
			}
		case "copy_tags":
			{
				go bot.copyTags(message)
				isRightCommand = true
			}
		default:
			{
				isRightCommand = false
			}
		}
	}

	return isRightCommand
}

// send отправляет сообщение
func (bot *Bot) send(msg tgbotapi.MessageConfig) {
	_, err := bot.botAPI.Send(msg)
	if err != nil {
		if err.Error() != "Forbidden: bot was blocked by the user" &&
			err.Error() != "Forbidden: user is deactivated" {
			text := fmt.Sprintf("UserID: %d", msg.ChatID)
			logging.LogMinorError("send", text, err)
		}
	}
}

// messageHandler – обёртка над bot.send()
// Отправляет сообщения раз в rate time.Duration
func (bot *Bot) sendWrapper(milliseconds uint64) {
	nanoseconds := milliseconds * 1000000
	rate := time.Duration(nanoseconds)
	limiter := time.Tick(rate)
	for message := range bot.messages {
		<-limiter
		go bot.send(message)
	}
}
