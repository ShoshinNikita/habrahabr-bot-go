package bot

import (
	"encoding/json"
	"errors"
	"io/ioutil" // чтение файлов
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anaskhan96/soup"   // html parser
	"github.com/mmcdole/gofeed"    // Rss parser
	"gopkg.in/telegram-bot-api.v4" // Telegram api
	"github.com/jasonlvhit/gocron" // Job Scheduling Package

	"config"
	"db"      // взаимодействие с базой данных
	"logging" // логгирование
)


// Bot надстрройка над tgbotapi.BotAPI
type Bot struct {
	botAPI *tgbotapi.BotAPI
}


// NewBot инициализирует бота
func NewBot() *Bot {
	var err error

	// Инициализация бота
	var bot Bot
	bot.botAPI, err = tgbotapi.NewBotAPI(config.Data.BotToken)
	if err != nil {
		logging.LogFatalError("NewBot", "вызов NewBotAPI()", err)
	}

	bot.botAPI.Buffer = 12 * 50

	return &bot
}


// StartPooling начинает перехватывать сообщения
func (bot *Bot) StartPooling() {
	// Главный цикл
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updateChannel, err := bot.botAPI.GetUpdatesChan(updateConfig)
	if err != nil {
		logging.LogFatalError("NewBot", "попытка получить GetUpdatesChan", err)
	}

	// Страрт рассылки
	go bot.mailout()
	// Старт рассылки лучших статей каждый день в 21:00
	gocron.Every(1).Day().At("21:00").Do(bot.mailoutBestArticles)
	gocron.Start()

	// Long pooling
	for update := range updateChannel {
		if update.Message == nil {
			continue
		} else if !bot.distributeMessages(update.Message) {
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


// Notify отправляет пользователям сообщение, полученное через сайт
func (bot *Bot) Notify(sMessage string) error {
	users, err := db.GetAllUsers()
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


// send отправляет сообщение
func (bot *Bot) send(msg tgbotapi.MessageConfig) {
	bot.botAPI.Send(msg)
}


// start отвечает на команду /start, создаёт запись о пользователе
func (bot *Bot) start(msg *tgbotapi.Message) {
	// Создание пользователя
	err := db.CreateUser(strconv.FormatInt(msg.Chat.ID, 10))
	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/start",
			AddInfo:  "попытка создать пользователя"}
		logging.LogErrorAndNotify(data, bot.botAPI)
		return
	}

	message := tgbotapi.NewMessage(msg.Chat.ID, "Привет, "+msg.Chat.UserName+"! Введи /help для справки")
	message.ReplyMarkup = createKeyboard()
	bot.send(message)
}


// startMailout включает рассылку
func (bot *Bot) startMailout(command userCommand) {
	msg := command.message
	site := command.site

	var err error
	if site == habr {
		err = db.StartMailout(strconv.FormatInt(msg.Chat.ID, 10), habr)
	} else if site == geek {
		err = db.StartMailout(strconv.FormatInt(msg.Chat.ID, 10), geek)
	}

	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/start_mailout",
			AddInfo:  "попытка включить рассылку для " + site}
		logging.LogErrorAndNotify(data, bot.botAPI)
		return
	}

	message := tgbotapi.NewMessage(msg.Chat.ID, "Рассылка для "+site+" включена")
	bot.send(message)
}


// stopMailout останавливает рассылку для пользователя
func (bot *Bot) stopMailout(command userCommand) {
	msg := command.message
	site := command.site

	var err error
	if site == habr {
		err = db.StopMailout(strconv.FormatInt(msg.Chat.ID, 10), habr)
	} else if site == geek {
		err = db.StopMailout(strconv.FormatInt(msg.Chat.ID, 10), geek)
	}
	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/...stop",
			AddInfo:  "попытка остановить рассылку для " + site}
		logging.LogErrorAndNotify(data, bot.botAPI)
		return
	}

	message := tgbotapi.NewMessage(msg.Chat.ID, "Рассылка приостановлена")
	bot.send(message)
}


// help отправляет справочную информацию
func (bot *Bot) help(msg *tgbotapi.Message) {
	message := tgbotapi.NewMessage(msg.Chat.ID, helpText)
	message.ParseMode = "HTML"
	bot.send(message)
}


// getStatus возвращает теги пользователя и информация, осуществляется ли рассылка
func (bot *Bot) getStatus(command userCommand) {
	msg := command.message
	site := command.site

	user, err := db.GetUser(strconv.FormatInt(msg.Chat.ID, 10))
	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/...tags",
			AddInfo:  "попытка получить данные пользователя"}
		logging.LogErrorAndNotify(data, bot.botAPI)
		return
	}

	var tags []string
	if site == habr {
		tags = user.HabrTags
	} else if site == geek {
		tags = user.GeekTags
	}

	var text string
	if len(tags) == 0 {
		text = "Список тегов пуст"
	} else {
		text = "Список тегов:\n* "
		text += strings.Join(tags, "\n* ")
	}

	text += "\n\n📬 Рассылка: "

	if site == habr {
		if user.HabrMailout {
			text += "осуществляется"
		} else {
			text += "не осуществляется"
		}
	} else if site == geek {
		if user.GeekMailout {
			text += "осуществляется"
		} else {
			text += "не осуществляется"
		}
	}

	message := tgbotapi.NewMessage(msg.Chat.ID, text)
	bot.send(message)
}


// addTags добавляет теги, которые прислал пользователь
func (bot *Bot) addTags(command userCommand) {
	msg := command.message
	site := command.site

	newTags := strings.Split(strings.ToLower(msg.CommandArguments()), " ")
	if len(newTags) == 0 {
		logging.SendErrorToUser("список тегов не может быть пустым", bot.botAPI, msg.Chat.ID)
		return
	}

	var updatedTags []string
	var err error
	if site == habr {
		updatedTags, err = db.AddUserTags(strconv.FormatInt(msg.Chat.ID, 10), habr, newTags)
	} else if site == geek {
		updatedTags, err = db.AddUserTags(strconv.FormatInt(msg.Chat.ID, 10), geek, newTags)
	}
	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/...add_tags",
			AddInfo:  "попытка добавить теги"}
		logging.LogErrorAndNotify(data, bot.botAPI)
		return
	}

	var text string
	if len(updatedTags) == 0 {
		text = "Список тегов пуст"
	} else {
		text = "Список тегов:\n* "
		text += strings.Join(updatedTags, "\n* ")
	}

	message := tgbotapi.NewMessage(msg.Chat.ID, text)
	bot.send(message)
}


// delTags удаляет теги, которые прислал пользователь
func (bot *Bot) delTags(command userCommand) {
	msg := command.message
	site := command.site

	tagsForDel := strings.Split(strings.ToLower(msg.CommandArguments()), " ")
	if len(tagsForDel) == 0 {
		logging.SendErrorToUser("список тегов не может быть пустым", bot.botAPI, msg.Chat.ID)
		return
	}

	var updatedTags []string
	var err error
	if site == habr {
		updatedTags, err = db.DelUserTags(strconv.FormatInt(msg.Chat.ID, 10), habr, tagsForDel)
	} else if site == geek {
		updatedTags, err = db.DelUserTags(strconv.FormatInt(msg.Chat.ID, 10), geek, tagsForDel)
	}
	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/...del_tags",
			AddInfo:  "попытка удалить теги"}
		logging.LogErrorAndNotify(data, bot.botAPI)
		return
	}

	var text string
	if len(updatedTags) == 0 {
		text = "Список тегов пуст"
	} else {
		text = "Список тегов:\n* "
		text += strings.Join(updatedTags, "\n* ")
	}

	message := tgbotapi.NewMessage(msg.Chat.ID, text)
	bot.send(message)
}


// delAllTags очищает список тегов пользователя
func (bot *Bot) delAllTags(command userCommand) {
	msg := command.message
	site := command.site

	var err error
	if site == habr {
		err = db.DelAllUserTags(strconv.FormatInt(msg.Chat.ID, 10), habr)
	} else if site == geek {
		err = db.DelAllUserTags(strconv.FormatInt(msg.Chat.ID, 10), habr)
	}
	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/...del_all_tags",
			AddInfo:  "попытка удалить теги"}
		logging.LogErrorAndNotify(data, bot.botAPI)
		return
	}

	message := tgbotapi.NewMessage(msg.Chat.ID, "Список тегов очищен")
	bot.send(message)
}


// copyTags копирует теги пользователя со страницы Habrahabr
func (bot *Bot) copyTags(command userCommand) {
	msg := command.message
	site := command.site

	userURL := msg.CommandArguments()
	var res bool
	switch site {
	case habr:
		{
			res, _ = regexp.MatchString(habrUserRegexPattern, userURL)
		}
	case geek:
		{
			res, _ = regexp.MatchString(geekUserRegexPattern, userURL)
		}
	}
	// Проверка ссылки, которую отправил пользователь
	if !res {
		logging.SendErrorToUser("неверный формат ссылки", bot.botAPI, msg.Chat.ID)
		return
	}

	// Загрузка сайта
	resp, err := soup.Get(userURL)
	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/...copy_tags",
			AddInfo:  "попытка загрузить сайт"}
		logging.LogErrorAndNotify(data, bot.botAPI)
		return
	}

	var userTags []string

	// Получение тегов
	doc := soup.HTMLParse(resp)
	tags := doc.FindAll("li", "rel", "hub-popover")
	for _, tagNode := range tags {
		res := tagNode.Find("a")
		tag := res.Text()
		tag = strings.ToLower(tag)
		tag = strings.Replace(tag, " ", "_", -1)
		userTags = append(userTags, tag)
	}
	// Получение Блогов компаний
	tags = doc.FindAll("div", "class", "media-obj__body media-obj__body_list-view list-snippet")
	for _, tagNode := range tags {
		res := tagNode.Find("a")

		tag := "Блог компании " + res.Text()
		tag = strings.ToLower(tag)
		tag = strings.Replace(tag, " ", "_", -1)
		userTags = append(userTags, tag)
	}

	if len(userTags) == 0 {
		logging.SendErrorToUser("было обнаружено 0 тегов. Должно быть больше", bot.botAPI, msg.Chat.ID)
		return
	}

	switch site {
		case habr: {
			err = db.UpdateTags(strconv.FormatInt(msg.Chat.ID, 10), habr, userTags)
		}
		case geek: {
			err = db.UpdateTags(strconv.FormatInt(msg.Chat.ID, 10), habr, userTags)
		}
	}
	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/...copy_tags",
			AddInfo:  "попытка перезаписать теги"}
		logging.LogErrorAndNotify(data, bot.botAPI)
		return
	}

	text := "Теги обновлены. Список тегов:\n* " + strings.Join(userTags, "\n* ")
	message := tgbotapi.NewMessage(msg.Chat.ID, text)
	bot.send(message)
}


// sendIV отправляет пользователю ссылку на статью, которую он прислал, в виде InstantView
func (bot *Bot) sendIV(command userCommand) {
	habrRegexpPattern, _ := regexp.Compile(habrArticleRegexPattern)
	geekRegexpPattern, _ := regexp.Compile(geekArticleRegexPattern)

	msg := command.message
	site := command.site

	var link, instantViewURL string
	// Если сообщение попало сюда, значит, ссылка точно есть
	switch site {
		case habr: {
			link = habrRegexpPattern.FindString(msg.Text)
			instantViewURL = formatString(habrInstantViewURL, map[string]string{"url": link})
		}
		case geek: {
			link = geekRegexpPattern.FindString(msg.Text)
			instantViewURL = formatString(geekInstantViewURL, map[string]string{"url": link})
		}
	}

	text := "<a href=\"" + instantViewURL + "\">InstantView</a>\n\n" +
			"<a href=\"" + link + "\">Перейти к статье</a>\n\n" +
			"<a href=\"" + link + "#comments\">Перейти к комментариям</a>"

	message := tgbotapi.NewMessage(msg.Chat.ID, text)
	message.ParseMode = "HTML"
	bot.send(message)
}


// getBest отправляет пользователю лучшие статьи за сегодняшний день.
// По-умолчанию – 5, если пользователь указал другое число - другое
func (bot *Bot) getBest(command userCommand) {
	msg := command.message
	site := command.site

	parser := gofeed.NewParser()
	var feed *gofeed.Feed

	var err error
	switch site {
	case habr:
		feed, err = parser.ParseURL(bestHabrArticlesURL)
	case geek:
		feed, err = parser.ParseURL(bestGeekArticlesURL)
	}
	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/...best",
			AddInfo:  "попытка распарсить RSS-ленту"}
		logging.LogErrorAndNotify(data, bot.botAPI)
		return
	}

	bestArticles := "<b>Лучшие статьи за этот день:</b>\n"
	limit := 5
	// Проверка, было ли задано другое количество статей
	if msg.CommandArguments() != "" {
		temp, err := strconv.Atoi(msg.CommandArguments())
		if err == nil && temp > 0 {
			limit = temp
		}
	}

	// Создание списка статей (в виде строки)
	for i, item := range feed.Items {
		if i >= limit {
			break
		}
		number := strconv.Itoa(i + 1)
		bestArticles += number + ") " + formatString("<a href='{link}'>{title}</a>", map[string]string{"link": item.Link, "title": item.Title}) + "\n"
	}

	message := tgbotapi.NewMessage(msg.Chat.ID, bestArticles)
	message.ParseMode = "HTML"
	message.DisableWebPagePreview = true
	bot.send(message)
}


// handleKeyboard включает клавиатуру
func (bot *Bot) showKeyboard(msg *tgbotapi.Message) {
	message := tgbotapi.NewMessage(msg.Chat.ID, "Клавиатура включена")
	message.ReplyMarkup = createKeyboard()
	bot.send(message)
}


// hideKeyboard выключает клавиатуру
func (bot *Bot) hideKeyboard(msg *tgbotapi.Message) {
	message := tgbotapi.NewMessage(msg.Chat.ID, "Клавиатура выключена")
	message.ReplyMarkup = tgbotapi.NewRemoveKeyboard(false)
	bot.send(message)
}


// mailoutBestArticles рассылает список лучших статей с Habrahabr и Geektimes пользователям,
// у которых активна рассылка статей с соответствующего ресурса
func (bot *Bot) mailoutBestArticles() {
	logging.LogEvent("Рассылка лучших статей")
	startTime := time.Now()

	users, err := db.GetAllUsers()
	if err != nil {
		logging.LogMinorError("mailoutBestArticles", "попытка получить список пользователей", err)
		return
	}
	var habrBestArticles, geekBestArticles string

	parser := gofeed.NewParser()
	var feed *gofeed.Feed
	// Создание списка лучших статей с Habrahabr
	feed, err = parser.ParseURL(bestHabrArticlesURL)
	if err != nil {
		logging.LogMinorError("mailoutBestArticles", "попытка распарсить RSS-ленту Habrahabr", err)
	} else {
		habrBestArticles = "<b>Лучшие статьи за этот день на Habrahabr:</b>\n"
		limit := 5

		// Создание списка статей (в виде строки)
		for i, item := range feed.Items {
			if i >= limit {
				break
			}
			number := strconv.Itoa(i + 1)
			habrBestArticles += number + ") " + formatString("<a href='{link}'>{title}</a>", map[string]string{"link": item.Link, "title": item.Title}) + "\n"
		}
	}

	// Создание списка лучших статей с Geektimes
	feed, err = parser.ParseURL(bestGeekArticlesURL)
	if err != nil {
		logging.LogMinorError("mailoutBestArticles", "попытка распарсить RSS-ленту Geektimes", err)
	} else {
		geekBestArticles = "<b>Лучшие статьи за этот день на Geektimes:</b>\n"
		limit := 5

		// Создание списка статей (в виде строки)
		for i, item := range feed.Items {
			if i >= limit {
				break
			}
			number := strconv.Itoa(i + 1)
			geekBestArticles += number + ") " + formatString("<a href='{link}'>{title}</a>", map[string]string{"link": item.Link, "title": item.Title}) + "\n"
		}
	}

	// wg используется для блокировки функции до выполнения всех goroutine и отправки всех статей
	var wg sync.WaitGroup
	// Проход по всем пользователям
	for _, user := range users {
		if user.HabrMailout && habrBestArticles != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()

				message := tgbotapi.NewMessage(user.ID, habrBestArticles)
				message.ParseMode = "HTML"
				message.DisableWebPagePreview = true
				bot.send(message)
			}()
		}
		if user.GeekMailout && geekBestArticles != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()

				message := tgbotapi.NewMessage(user.ID, geekBestArticles)
				message.ParseMode = "HTML"
				message.DisableWebPagePreview = true
				bot.send(message)
			}()
		}
	}

	// Ждём отправки всех сообщений
	wg.Wait()
	logging.LogEvent("Рассылка лучших статей завершена. Время выполнения: " + time.Since(startTime).String())
}


// mailout рассылает статьи с периодичностью config.Delay наносекунд
func (bot *Bot) mailout() {
	var lastTime LastArticlesTime

	// Чтение LastTime
	raw, err := ioutil.ReadFile(config.Data.Prefix + "data/lastArticleTime.json")
	if err != nil {
		logging.LogFatalError("Mailout", "попытка прочесть lastArticleTime.json", err)
	}
	json.Unmarshal(raw, &lastTime)

	// Таймер
	ticker := time.NewTicker(time.Duration(config.Data.Delay))

	// Первый раз статьи отправляются сразу
	for ; true; <-ticker.C {
		allUsers, err := db.GetAllUsers()
		if err != nil {
			logging.LogMinorError("mailout", "ошибка при попытке получить список всех пользователей", err)
			continue
		}

		// Создание списка пользователей, которым нужно отправлять статьи
		var habrUsers, geekUsers []db.User
		for _, user := range allUsers {
			if user.HabrMailout {
				habrUsers = append(habrUsers, user)
			}
			if user.GeekMailout {
				geekUsers = append(geekUsers, user)
			}
		}

		// Рассылка статей с Habrahabr
		logging.LogEvent("Рассылка статей с Habrahabr")
		startTime := time.Now()
		err = habrMailout(bot, habrUsers, &lastTime)
		if err != nil {
			logging.LogMinorError("habrMailout", "вызов habrMailout", err)
		}
		logging.LogEvent("Завершена. Время выполнения: " + time.Since(startTime).String())

		time.Sleep(time.Second * 1)

		// Рассылка статей с Geektimes
		logging.LogEvent("Рассылка статей с Geektimes")
		startTime = time.Now()
		err = geekMailout(bot, geekUsers, &lastTime)
		if err != nil {
			logging.LogMinorError("geekMailout", "вызов geekMailout", err)
		}
		logging.LogEvent("Завершена. Время выполнения: " + time.Since(startTime).String())

		// Перезапись времени
		raw, _ = json.Marshal(lastTime)
		err = ioutil.WriteFile(config.Data.Prefix+"data/lastArticleTime.json", raw, 0644)
		if err != nil {
			logging.LogFatalError("Mailout", "попытка записать файл lastArticleTime.json", err)
		}

	}
}


// habrMailout отвечает за рассылку статей с сайта Habrahabr.ru
func habrMailout(bot *Bot, allUsers []db.User, lastTime *LastArticlesTime) error {
	// Parser
	parser := gofeed.NewParser()

	// Получение RSS-ленты
	feed, err := parser.ParseURL(allHabrArticlesURL)
	if err != nil {
		return err
	}

	// Создание списка новых статей
	var newArticles []article
	for _, item := range feed.Items {
		articleTime, err := time.Parse(time.RFC1123, item.Published)
		if err != nil {
			logging.LogMinorError("Mailout", "", err)
			continue
		}
		// Проверка, была ли статья опубликована позже, чем была последняя проверка RSS-ленты
		if lastTime.Habr < articleTime.Unix() {
			// Создание списка тегов статьи
			var tags []string
			for _, tag := range item.Categories {
				// Форматирование от "Some Tag" к "some_tag"
				tag = strings.Replace(tag, " ", "_", -1)
				tag = strings.ToLower(tag)
				tags = append(tags, tag)
			}
			instantView := formatString(habrInstantViewURL, map[string]string{"url": item.Link})
			message := formatString(messageText, map[string]string{"source": "Habrahabr", "title": item.Title, "IV": instantView, "link": item.Link})

			temp := article{title: item.Title, tags: tags, link: item.Link, message: message}

			newArticles = append(newArticles, temp)
		} else {
			break
		}
	}

	// Если новых статей не было, то отправлять статьи и обновлять время не нужно
	if len(newArticles) == 0 {
		return nil
	}

	// Отправка статей
	// Проход по всем пользователям
	articlesCounter := 0
	for _, user := range allUsers {
		// Проход по всем статьям в обратном порядке
		for i := len(newArticles) - 1; i >= 0; i-- {
			shouldSend := false
			if len(user.HabrTags) == 0 {
				shouldSend = true
			} else {
				// Проверка, есть ли теги пользователя в статье
				for _, tag := range newArticles[i].tags {
					for _, userTag := range user.HabrTags {
						if tag == userTag {
							shouldSend = true
							goto send
						}
					}
				}
			}
		send:

			// Отправка пользователю
			if shouldSend {
				articlesCounter++
				message := tgbotapi.NewMessage(user.ID, newArticles[i].message)
				message.ParseMode = "HTML"

				t := time.Now()

				bot.send(message)

				since := time.Since(t)
				if since >= time.Second * 1 {
					logging.LogMinorError("habrMailout", "Отправка статьи заняла " + since.String(), errors.New(""))
				}
			}
		}
	}

	// Обновление времени
	tempTime, err := time.Parse(time.RFC1123, feed.Items[0].Published)
	if err != nil {
		return err
	}
	lastTime.Habr = tempTime.Unix()

	return nil
}


// geekMailout отвечает за рассылку статей с сайта Geektimes.ru
func geekMailout(bot *Bot, allUsers []db.User, lastTime *LastArticlesTime) error {
	// Parser
	parser := gofeed.NewParser()

	// Получение RSS-ленты
	feed, err := parser.ParseURL(allGeekArticlesURL)
	if err != nil {
		return err
	}

	// Создание списка новых статей
	var newArticles []article
	for _, item := range feed.Items {
		articleTime, err := time.Parse(time.RFC1123, item.Published)
		if err != nil {
			logging.LogMinorError("Mailout", "", err)
			continue
		}
		// Проверка, была ли статья опубликована позже, чем была последняя проверка RSS-ленты
		if lastTime.Geek < articleTime.Unix() {
			// Создание списка тегов статьи
			var tags []string
			for _, tag := range item.Categories {
				// Форматирование от "Some Tag" к "some_tag"
				tag = strings.Replace(tag, " ", "_", -1)
				tag = strings.ToLower(tag)
				tags = append(tags, tag)
			}
			instantView := formatString(geekInstantViewURL, map[string]string{"url": item.Link})
			message := formatString(messageText, map[string]string{"source": "Geektimes", "title": item.Title, "IV": instantView, "link": item.Link})

			temp := article{title: item.Title, tags: tags, link: item.Link, message: message}

			newArticles = append(newArticles, temp)
		} else {
			break
		}
	}

	// Если новых статей не было, то отправлять статьи и обновлять время не нужно
	if len(newArticles) == 0 {
		return nil
	}

	// Отправка статей
	// Проход по всем пользователям
	for _, user := range allUsers {
		// Проход по всем статьям в обратном порядке
		for i := len(newArticles) - 1; i >= 0; i-- {

			shouldSend := false
			if len(user.GeekTags) == 0 {
				shouldSend = true
			} else {
				// Проверка, есть ли теги пользователя в статье
				for _, tag := range newArticles[i].tags {
					for _, userTag := range user.GeekTags {
						if tag == userTag {
							shouldSend = true
							goto send
						}
					}
				}
			}
		send:

			// Отправка пользователю
			if shouldSend {
				message := tgbotapi.NewMessage(user.ID, newArticles[i].message)
				message.ParseMode = "HTML"

				t := time.Now()

				bot.send(message)

				since := time.Since(t)
				if since >= time.Second * 1 {
					logging.LogMinorError("geekMailout", "Отправка статьи заняла " + since.String(), errors.New(""))
				}
			}
		}
	}

	// Обновление времени
	tempTime, err := time.Parse(time.RFC1123, feed.Items[0].Published)
	if err != nil {
		return err
	}
	lastTime.Geek = tempTime.Unix()

	return nil
}