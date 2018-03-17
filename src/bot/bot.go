package bot


import (
	"database/sql" // database
	"encoding/json"
	"io/ioutil" // чтение файлов
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/anaskhan96/soup"	// html parser
	_ "github.com/mattn/go-sqlite3" // sql
	"github.com/mmcdole/gofeed" 	// Rss parser
	"gopkg.in/telegram-bot-api.v4"  // Telegram api

	"logging"
)


// Bot надстрройка над tgbotapi.BotAPI
type Bot struct {
	botAPI *tgbotapi.BotAPI
	config ConfigData
	db     *sql.DB

	// Каналы
	startChan       chan userCommand
	helpChan        chan *tgbotapi.Message
	stopMailoutChan chan userCommand
	getTagsChan     chan userCommand
	addTagsChan     chan userCommand
	delTagsChan     chan userCommand
	delAllTagsChan  chan userCommand
	copyTagsChan    chan userCommand
	sendIVChan      chan userCommand
	getBestChan     chan userCommand
}


// NewBot инициализирует бота
func NewBot(config ConfigData) *Bot {
	var err error

	// Инициализация бота
	var bot Bot
	bot.config = config
	bot.botAPI, err = tgbotapi.NewBotAPI(bot.config.Token)
	if err != nil {
		logging.LogFatalError("main", err)
	}

	// Инициализация SQLite db
	db, err := sql.Open("sqlite3", "data/database.db")
	if err != nil {
		logging.LogFatalError("main", err)
	}
	bot.db = db

	bot.botAPI.Buffer = 12 * 50

	// Инициализация каналов
	bot.startChan = 		make(chan userCommand, 50)
	bot.helpChan =  		make(chan *tgbotapi.Message, 50)
	bot.stopMailoutChan =   make(chan userCommand, 50)
	bot.getTagsChan = 		make(chan userCommand, 50)
	bot.addTagsChan = 		make(chan userCommand, 50)
	bot.delTagsChan = 		make(chan userCommand, 50)
	bot.delAllTagsChan = 	make(chan userCommand, 50)
	bot.copyTagsChan = 		make(chan userCommand, 50)
	bot.sendIVChan = 		make(chan userCommand, 50)
	bot.getBestChan = 		make(chan userCommand, 50)

	return &bot
}


// StartPooling начинает перехватывать сообщения
func (bot *Bot) StartPooling() {
	// Goroutines
	go bot.start(bot.startChan)
	go bot.help(bot.helpChan)
	go bot.stopMailoutForUser(bot.stopMailoutChan)
	go bot.mailout(bot.config)
	go bot.returnTags(bot.getTagsChan)
	go bot.addTags(bot.addTagsChan)
	go bot.delTags(bot.delTagsChan)
	go bot.delAllTags(bot.delAllTagsChan)
	go bot.getBest(bot.getBestChan, bot.config)
	go bot.copyTags(bot.copyTagsChan)
	go bot.sendIV(bot.sendIVChan, bot.config)

	// Главный цикл
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updateChannel, err := bot.botAPI.GetUpdatesChan(updateConfig)
	if err != nil {
		logging.LogFatalError("main", err)
	}

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
			bot.sendIVChan <- userCommand{message, habr}
			isRightCommand = true
		} else if res, _ = regexp.MatchString(geekArticleRegexPattern, message.Text); res {
			bot.sendIVChan <- userCommand{message, geek}
			isRightCommand = true
		}
	} else {
		// Рассматривается отдельно, т.к. команда используется без префиксов
		if command == "help" {
			bot.helpChan <- message
			return true
		}
		// Если команда == /start, то site==""
		if command != "start" {
			// Длина всегда > 5
			if len(command) <= 5 {
				return false
			}
			if prefix := command[:5]; prefix == "geek_" {
				site = geek
			} else if prefix == "habr_" {
				site = habr
			}
			command = command[5:]
		}
		switch command {
			case "start": {
				bot.startChan <- userCommand{message, site}
				isRightCommand = true
			}
			case "stop": {
				bot.stopMailoutChan <- userCommand{message, site}
				isRightCommand = true
			}
			case "tags": {
				bot.getTagsChan <- userCommand{message, site}
				isRightCommand = true
			}
			case "add_tags": {
				bot.addTagsChan <- userCommand{message, site}
				isRightCommand = true
			}
			case "del_tags":
			{
				bot.delTagsChan <- userCommand{message, site}
				isRightCommand = true
			}
			case "del_all_tags": {
				bot.delAllTagsChan <- userCommand{message, site}
				isRightCommand = true
			}
			case "best": {
				bot.getBestChan <- userCommand{message, site}
				isRightCommand = true
			}
			case "copy_tags": {
				bot.copyTagsChan <- userCommand{message, site}
				isRightCommand = true
			}
		}
	}

	return isRightCommand
}


// Notify отправляет пользователям сообщение, полученное через сайт
func (bot *Bot) Notify(sMessage string) {
	var counter int // Следит за тем, чтобы не было отправлено больше maxArticlesLimit сообщений в секунду

	rows, err := bot.db.Query(`SELECT id FROM users`)
	if err != nil {
		logging.LogMinorError("Notify", err)
		return
	}
	var id int64
	for rows.Next() {
		rows.Scan(&id)
		message := tgbotapi.NewMessage(id, sMessage)
		message.ParseMode = "HTML"
		bot.send(message)

		counter++
		if counter >= maxArticlesLimit {
			counter = 0
			time.Sleep(time.Second)
		}
	}
}


// send отправляет сообщение
func (bot *Bot) send(msg tgbotapi.MessageConfig) (tgbotapi.Message, error) {
	return bot.botAPI.Send(msg)
}


// start отвечает на команду /start, создаёт запись о пользователе
func (bot *Bot) start(data chan userCommand) {
	startMailout := func(tx *sql.Tx, id int64) (sql.Result, error) {
		return tx.Exec(`UPDATE users SET habr_is_stop=0, geek_is_stop=0 WHERE id=?`, id)
	}
	startHabrMailout := func(tx *sql.Tx, id int64) (sql.Result, error) {
		return tx.Exec(`UPDATE users SET habr_is_stop=0 WHERE id=?`, id)
	}
	startGeekMailout := func(tx *sql.Tx, id int64) (sql.Result, error) {
		return tx.Exec(`UPDATE users SET geek_is_stop=0 WHERE id=?`, id)
	}

	var msg *tgbotapi.Message
	var site string

	for command := range data {
		msg = command.message
		site = command.site

		tx, err := bot.db.Begin()
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		// Создание пользователя
		_, err = tx.Exec(`INSERT OR IGNORE INTO users(id) VALUES(?)`, msg.Chat.ID)
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		switch site {
		case "":
			_, err = startMailout(tx, msg.Chat.ID)
		case habr:
			_, err = startHabrMailout(tx, msg.Chat.ID)
		case geek:
			_, err = startGeekMailout(tx, msg.Chat.ID)
		}
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		tx.Commit()

		message := tgbotapi.NewMessage(msg.Chat.ID, "Привет, " + msg.Chat.UserName + "! Введи /help для справки")
		_, err = bot.send(message)
		if err != nil {
			logging.LogSendingError(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID})
		}
	}
}


// help отправляет справочную информацию
func (bot *Bot) help(data chan *tgbotapi.Message) {
	for msg := range data {
		message := tgbotapi.NewMessage(msg.Chat.ID, helpText)
		message.ParseMode = "HTML"
		_, err := bot.send(message)
		if err != nil {
			logging.LogSendingError(logging.ErrorData{Err: err, Username: msg.Chat.UserName, UserID: msg.Chat.ID})
		}
	}
}


// returnTags возвращает теги пользователя
func (bot *Bot) returnTags(data chan userCommand) {
	getHabrTags := func(id int64) (*sql.Rows, error) {
		return bot.db.Query(`SELECT habr_tags FROM users WHERE id=?`, id)
	}
	getGeekTags := func(id int64) (*sql.Rows, error) {
		return bot.db.Query(`SELECT geek_tags FROM users WHERE id=?`, id)
	}
	var rows *sql.Rows
	var err error
	var msg *tgbotapi.Message
	var site string

	for command := range data {
		msg = command.message
		site = command.site
		switch site {
		case habr:
			rows, err = getHabrTags(msg.Chat.ID)
		case geek:
			rows, err = getGeekTags(msg.Chat.ID)
		}

		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		var tags string
		for rows.Next() {
			rows.Scan(&tags)
		}
		rows.Close()

		var text string
		if tags == "" {
			text = "Список тегов пуст"
		} else {
			text = "Список тегов:\n* " + strings.Replace(tags, " ", "\n* ", -1)
		}

		message := tgbotapi.NewMessage(msg.Chat.ID, text)
		_, err = bot.send(message)
		if err != nil {
			logging.LogSendingError(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID})
		}
	}
}


// addTags добавляет теги, которые прислал пользователь
func (bot *Bot) addTags(data chan userCommand) {
	getHabrTags := func(id int64) (*sql.Rows, error) {
		return bot.db.Query(`SELECT habr_tags FROM users WHERE id=?`, id)
	}
	getGeekTags := func(id int64) (*sql.Rows, error) {
		return bot.db.Query(`SELECT geek_tags FROM users WHERE id=?`, id)
	}

	changeHabrTags := func(tx *sql.Tx, newTags string, id int64) (sql.Result, error) {
		return tx.Exec(`UPDATE users SET habr_tags=? WHERE id=?`, newTags, id)
	}
	changeGeekTags := func(tx *sql.Tx, newTags string, id int64) (sql.Result, error) {
		return tx.Exec(`UPDATE users SET geek_tags=? WHERE id=?`, newTags, id)
	}

	var rows *sql.Rows
	var err error
	var msg *tgbotapi.Message
	var site string

	for command := range data {
		msg = command.message
		site = command.site

		if msg.CommandArguments() == "" {
			logging.SendErrorToUser("список тегов не может быть пустым", bot.botAPI, msg.Chat.ID)
			continue
		}

		switch site {
		case habr:
			rows, err = getHabrTags(msg.Chat.ID)
		case geek:
			rows, err = getGeekTags(msg.Chat.ID)
		}
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}

		var strOldTags string
		for rows.Next() {
			rows.Scan(&strOldTags)
		}
		rows.Close()

		newTags := getTagsFromString(strings.ToLower(msg.CommandArguments()))
		userTags := getTagsFromString(strOldTags)

		for k := range newTags {
			userTags[k] = true
		}

		var strUserTags string // Обновлённые теги
		for k := range userTags {
			strUserTags += k + " "
		}
		strUserTags = strings.TrimSuffix(strUserTags, " ")

		tx, err := bot.db.Begin()
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		switch site {
		case habr:
			_, err = changeHabrTags(tx, strUserTags, msg.Chat.ID)
		case geek:
			_, err = changeGeekTags(tx, strUserTags, msg.Chat.ID)
		}
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		tx.Commit()

		var text string
		if strUserTags == "" {
			text = "Список тегов пуст"
		} else {
			text = "Теги обновлены. Список тегов:\n* " + strings.Replace(strUserTags, " ", "\n* ", -1)
		}

		message := tgbotapi.NewMessage(msg.Chat.ID, text)
		_, err = bot.send(message)
		if err != nil {
			logging.LogSendingError(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID})
		}
	}
}


// delTags удаляет теги, которые прислал пользователь
func (bot *Bot) delTags(data chan userCommand) {
	getHabrTags := func(id int64) (*sql.Rows, error) {
		return bot.db.Query(`SELECT habr_tags FROM users WHERE id=?`, id)
	}
	getGeekTags := func(id int64) (*sql.Rows, error) {
		return bot.db.Query(`SELECT geek_tags FROM users WHERE id=?`, id)
	}

	changeHabrTags := func(tx *sql.Tx, newTags string, id int64) (sql.Result, error) {
		return tx.Exec(`UPDATE users SET habr_tags=? WHERE id=?`, newTags, id)
	}
	changeGeekTags := func(tx *sql.Tx, newTags string, id int64) (sql.Result, error) {
		return tx.Exec(`UPDATE users SET geek_tags=? WHERE id=?`, newTags, id)
	}

	var rows *sql.Rows
	var err error
	var msg *tgbotapi.Message
	var site string

	for command := range data {
		msg = command.message
		site = command.site

		if msg.CommandArguments() == "" {
			logging.SendErrorToUser("список тегов не может быть пустым", bot.botAPI, msg.Chat.ID)
			continue
		}
		switch site {
		case habr:
			rows, err = getHabrTags(msg.Chat.ID)
		case geek:
			rows, err = getGeekTags(msg.Chat.ID)
		}
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}

		var strOldTags string
		for rows.Next() {
			rows.Scan(&strOldTags)
		}
		rows.Close()

		tagsForDel := getTagsFromString(strings.ToLower(msg.CommandArguments()))
		userTags := getTagsFromString(strOldTags)
		for k := range tagsForDel {
			delete(userTags, k)
		}

		var strUserTags string // Обновлённые теги
		for k := range userTags {
			strUserTags += k + " "
		}
		strUserTags = strings.TrimSuffix(strUserTags, " ")

		tx, err := bot.db.Begin()
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		switch site {
		case habr:
			_, err = changeHabrTags(tx, strUserTags, msg.Chat.ID)
		case geek:
			_, err = changeGeekTags(tx, strUserTags, msg.Chat.ID)
		}
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		tx.Commit()

		var text string
		if strUserTags == "" {
			text = "Список тегов пуст"
		} else {
			text = "Теги обновлены. Список тегов:\n* " + strings.Replace(strUserTags, " ", "\n* ", -1)
		}

		message := tgbotapi.NewMessage(msg.Chat.ID, text)
		_, err = bot.send(message)
		if err != nil {
			logging.LogSendingError(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID})
		}
	}
}


// delAllTags очищает список тегов пользователя
func (bot *Bot) delAllTags(data chan userCommand) {
	delHabrTags := func(tx *sql.Tx, id int64) (sql.Result, error) {
		return tx.Exec(`UPDATE users SET habr_tags='' WHERE id=?`, id)
	}
	delGeekTags := func(tx *sql.Tx, id int64) (sql.Result, error) {
		return tx.Exec(`UPDATE users SET geek_tags='' WHERE id=?`, id)
	}

	var msg *tgbotapi.Message
	var site string

	for command := range data {
		msg = command.message
		site = command.site
		tx, err := bot.db.Begin()
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		switch site {
		case habr:
			_, err = delHabrTags(tx, msg.Chat.ID)
		case geek:
			_, err = delGeekTags(tx, msg.Chat.ID)
		}
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		tx.Commit()

		message := tgbotapi.NewMessage(msg.Chat.ID, "Список тегов очищен")
		_, err = bot.send(message)
		if err != nil {
			logging.LogSendingError(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID})
		}
	}
}


// copyTags копирует теги пользователя со страницы Habrahabr
func (bot *Bot) copyTags(data chan userCommand) {
	changeHabrTags := func(tx *sql.Tx, newTags string, id int64) (sql.Result, error) {
		return tx.Exec(`UPDATE users SET habr_tags=? WHERE id=?`, newTags, id)
	}
	changeGeekTags := func(tx *sql.Tx, newTags string, id int64) (sql.Result, error) {
		return tx.Exec(`UPDATE users SET geek_tags=? WHERE id=?`, newTags, id)
	}

	var msg *tgbotapi.Message
	var site string

	for command := range data {
		msg = command.message
		site = command.site

		userURL := msg.CommandArguments()
		var res bool
		switch site {
			case habr: {
				res, _ = regexp.MatchString(habrUserRegexPattern, userURL)
			}
			case geek: {
				res, _ = regexp.MatchString(geekUserRegexPattern, userURL)
			}
		}
		// Проверка ссылки, которую отправил пользователь
		if !res {
			logging.SendErrorToUser("неверный формат ссылки", bot.botAPI, msg.Chat.ID)
			continue
		}

		// Загрузка сайта
		resp, err := soup.Get(userURL)
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
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
			continue
		}
		strUserTags := strings.Join(userTags, " ")

		tx, err := bot.db.Begin()
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		switch site {
			case habr: {
				_, err = changeHabrTags(tx, strUserTags, msg.Chat.ID)
			}
			case geek: {
				_, err = changeGeekTags(tx, strUserTags, msg.Chat.ID)
			}
		}
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		tx.Commit()

		text := "Теги обновлены. Список тегов:\n* " + strings.Replace(strUserTags, " ", "\n* ", -1)

		message := tgbotapi.NewMessage(msg.Chat.ID, text)
		_, err = bot.send(message)
		if err != nil {
			logging.LogSendingError(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID})
		}
	}
}


// stopMailoutForUser останавливает рассылку для пользователя
func (bot *Bot) stopMailoutForUser(data chan userCommand) {
	stopHabrMailout := func(tx *sql.Tx, id int64) (sql.Result, error) {
		return tx.Exec(`UPDATE users SET habr_is_stop=1 WHERE id=?`, id)
	}
	stopGeekMailout := func(tx *sql.Tx, id int64) (sql.Result, error) {
		return tx.Exec(`UPDATE users SET geek_is_stop=1 WHERE id=?`, id)
	}

	var msg *tgbotapi.Message
	var site string

	for command := range data {
		msg = command.message
		site = command.site

		tx, err := bot.db.Begin()
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		switch site {
		case habr:
			_, err = stopHabrMailout(tx, msg.Chat.ID)
		case geek:
			_, err = stopGeekMailout(tx, msg.Chat.ID)
		}
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		tx.Commit()

		message := tgbotapi.NewMessage(msg.Chat.ID, "Рассылка приостановлена")
		_, err = bot.send(message)
		if err != nil {
			logging.LogSendingError(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID})
		}
	}
}


// sendIV отправляет пользователю ссылку на статью, которую он прислал, в виде InstantView
func (bot *Bot) sendIV(data chan userCommand, config ConfigData) {
	habrRegexpPattern, _ := regexp.Compile(habrArticleRegexPattern)
	geekRegexpPattern, _ := regexp.Compile(geekArticleRegexPattern)

	var msg *tgbotapi.Message
	var site string

	for command := range data {
		msg = command.message
		site = command.site
		var link, instantViewURL string
		
		// Если сообщение попало сюда, значит, ссылка точно есть
		switch site {
			case habr: {
				link = habrRegexpPattern.FindString(msg.Text)
				instantViewURL = formatString(habrInstantViewURL, map[string]string{"url": link})
			}
			case geek:{
				link = geekRegexpPattern.FindString(msg.Text)
				instantViewURL = formatString(geekInstantViewURL, map[string]string{"url": link})
			}
		}

		text := "<a href=\"" + instantViewURL + "\">InstantView</a>\n\n" +
			"<a href=\"" + link + "\">Перейти к статье</a>\n\n" +
			"<a href=\"" + link + "#comments\">Перейти к комментариям</a>"

		message := tgbotapi.NewMessage(msg.Chat.ID, text)
		message.ParseMode = "HTML"
		_, err := bot.send(message)
		if err != nil {
			logging.LogSendingError(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID})
		}
	}
}


// getBest отправляет пользователю лучшие статьи за сегодняшний день.
// По-умолчанию – 5, если пользователь указал другое число - другое
func (bot *Bot) getBest(data chan userCommand, config ConfigData) {
	parser := gofeed.NewParser()

	var msg *tgbotapi.Message
	var site string

	const link string = "<a href='{link}'>{title}</a>"

	for command := range data {
		msg = command.message
		site = command.site
		var feed *gofeed.Feed
		var err error
		switch site {
		case habr:
			feed, err = parser.ParseURL(bestHabrArticlesURL)
		case geek:
			feed, err = parser.ParseURL(bestGeekArticlesURL)
		}
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
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
			bestArticles += number + ") " + formatString(link, map[string]string{"link": item.Link, "title": item.Title}) + "\n"
		}

		message := tgbotapi.NewMessage(msg.Chat.ID, bestArticles)
		message.ParseMode = "HTML"
		_, err = bot.send(message)
		if err != nil {
			logging.LogSendingError(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID})
		}
	}
}


// mailout рассылает статьи с периодичностью config.Delay наносекунд
func (bot *Bot) mailout(config ConfigData) {
	var lastTime LastArticlesTime

	// Чтение LastTime
	raw, err := ioutil.ReadFile("./data/lastArticleTime.json")
	if err != nil {
		logging.LogFatalError("Mailout", err)
	}
	json.Unmarshal(raw, &lastTime)

	// Таймер
	ticker := time.NewTicker(time.Duration(config.Delay))

	// Первый раз статьи отправляются сразу
	for ; true; <-ticker.C {
		logging.LogEvent("Рассылка статей с Habrahabr")
		err = habrMailout(bot, &lastTime)
		if err != nil {
			logging.LogMinorError("habrMailout", err)
		}

		logging.LogEvent("Рассылка статей с Geektimes")
		err = geekMailout(bot, &lastTime)
		if err != nil {
			logging.LogMinorError("geekMailout", err)
		}

		// Перезапись времени
		raw, err = json.Marshal(lastTime)
		if err != nil {
			logging.LogMinorError("Mailout", err)
		} else {
			err = ioutil.WriteFile("./data/lastArticleTime.json", raw, 0644)
			if err != nil {
				logging.LogFatalError("Mailout", err)
			}
		}
	}
}


// habrMailout отвечает за рассылку статей с сайта Habrahabr.ru
func habrMailout(bot *Bot, lastTime *LastArticlesTime) error {
	// Подсчитывает количество отправленных статей за 1 цикл рассылки. Если превышает 25, то бот засыпает на 1 с
	counterSendedArticles := 0

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
			logging.LogMinorError("Mailout", err)
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
			message := formatString(messageText, 
									map[string]string{"source": "Habrahabr", "title": item.Title, "IV": instantView, "link": item.Link})

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
	// Получение списка пользователей, которым можно отправлять статьи
	users, err := bot.db.Query(`SELECT id, habr_tags FROM users WHERE habr_is_stop=0`)
	if err != nil {
		return err
	}
	// Проход по всем пользователям
	for users.Next() {
		var id int64
		var sTags string
		users.Scan(&id, &sTags)
		var userTags []string
		if sTags != "" {
			userTags = strings.Split(sTags, " ")
		}

		// Проход по всем статьям в обратном порядке
		for i := len(newArticles) - 1; i >= 0; i-- {
			shouldSend := false
			if len(userTags) == 0 {
				shouldSend = true
			} else {
				// Проверка, есть ли теги пользователя в статье
				for _, tag := range newArticles[i].tags {
					for _, userTag := range userTags {
						if tag == userTag {
							shouldSend = true
							goto loopEnd
						}
					}
				}
			loopEnd:
			}
			// Отправка пользователю
			if shouldSend {
				counterSendedArticles++

				message := tgbotapi.NewMessage(id, newArticles[i].message)
				message.ParseMode = "HTML"
				_, err = bot.send(message)
				if err != nil {
					logging.LogSendingError(logging.ErrorData{err, "Empty", id})
				}
			}

			// Если превышено количество отправленных статей, то goroutine засыпает на 1 секунду
			if counterSendedArticles > maxArticlesLimit {
				counterSendedArticles = 0
				time.Sleep(time.Second)
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
func geekMailout(bot *Bot, lastTime *LastArticlesTime) error {
	// Подсчитывает количество отправленных статей за 1 цикл рассылки. Если превышает 25, то бот засыпает на 1 с
	counterSendedArticles := 0

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
			logging.LogMinorError("Mailout", err)
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
			message := formatString(messageText, 
								map[string]string{"source": "Geektimes", "title": item.Title, "IV": instantView, "link": item.Link})

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
	// Получение списка пользователей, которым можно отправлять статьи
	users, err := bot.db.Query(`SELECT id, geek_tags FROM users WHERE geek_is_stop=0`)
	if err != nil {
		return err
	}
	// Проход по всем пользователям
	for users.Next() {
		var id int64
		var sTags string
		users.Scan(&id, &sTags)
		var userTags []string
		if sTags != "" {
			userTags = strings.Split(sTags, " ")
		}

		// Проход по всем статьям в обратном порядке
		for i := len(newArticles) - 1; i >= 0; i-- {
			shouldSend := false
			if len(userTags) == 0 {
				shouldSend = true
			} else {
				// Проверка, есть ли теги пользователя в статье
				for _, tag := range newArticles[i].tags {
					for _, userTag := range userTags {
						if tag == userTag {
							shouldSend = true
							goto loopEnd
						}
					}
				}
			loopEnd:
			}
			// Отправка пользователю
			if shouldSend {
				counterSendedArticles++

				message := tgbotapi.NewMessage(id, newArticles[i].message)
				message.ParseMode = "HTML"
				_, err = bot.send(message)
				if err != nil {
					logging.LogSendingError(logging.ErrorData{err, "Empty", id})
				}
			}

			// Если превышено количество отправленных статей, то goroutine засыпает на 1 секунду
			if counterSendedArticles > maxArticlesLimit {
				counterSendedArticles = 0
				time.Sleep(time.Second)
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
