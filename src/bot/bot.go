package bot

import (
	//"log"
	"fmt"
	"strings"
	"strconv"
	"io/ioutil" // чтение файлов
	"database/sql" // database
	"time"
	"regexp"

	"gopkg.in/telegram-bot-api.v4" // Telegram api
	_ "github.com/mattn/go-sqlite3" // sql
	"github.com/mmcdole/gofeed" // Rss parser
	"github.com/anaskhan96/soup" // html parser

	"logging"
)


// HabrahabrBot надстрройка над tgbotapi.BotAPI
type HabrahabrBot struct {
	botAPI *tgbotapi.BotAPI
	config ConfigData
	db *sql.DB
}


// NewBot инициализирует бота
func NewBot(config ConfigData) (*HabrahabrBot) {
	var err error

	// Инициализация бота
	var bot HabrahabrBot
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

	bot.botAPI.Buffer = 11 * 50
	
	return &bot
}


// StartPooling начинает перехватывать сообщения
func (bot *HabrahabrBot) StartPooling() {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	// Каналы
	updateChannel, err := bot.botAPI.GetUpdatesChan(updateConfig)
	if err != nil {
		logging.LogFatalError("main", err)
	}
	startChan := make(chan *tgbotapi.Message, 50)
	helpChan := make(chan *tgbotapi.Message, 50)
	getTagsChan := make(chan *tgbotapi.Message, 50)
	addTagsChan := make(chan *tgbotapi.Message, 50)
	delTagsChan := make(chan *tgbotapi.Message, 50)
	delAllTagsChan := make(chan *tgbotapi.Message, 50)
	stopMailoutChan := make(chan *tgbotapi.Message, 50)
	copyTagsChan := make(chan *tgbotapi.Message, 50)
	sendIVChan := make(chan *tgbotapi.Message, 50)
	getBestChan := make(chan *tgbotapi.Message, 50)

	// Goroutines
	go bot.start(startChan)
	go bot.help(helpChan)
	go bot.getTags(getTagsChan)
	go bot.addTags(addTagsChan)
	go bot.delTags(delTagsChan)
	go bot.delAllTags(delAllTagsChan)
	go bot.stopMailoutForUser(stopMailoutChan)
	go bot.copyTags(copyTagsChan)
	go bot.sendIV(sendIVChan, bot.config)
	go bot.mailout(bot.config)
	go bot.getBest(getBestChan, bot.config)

	// Главный цикл
	for update := range updateChannel {
		if update.Message == nil {
			continue
		} else if update.Message.Command() == "start" {
			startChan <- update.Message
		} else if update.Message.Command() == "help" {
			helpChan <- update.Message
		} else if update.Message.Command() == "my_tags" {
			getTagsChan <- update.Message
		} else if update.Message.Command() == "add_tags" {
			addTagsChan <- update.Message
		} else if update.Message.Command() == "del_tags" {
			delTagsChan <- update.Message
		} else if update.Message.Command() == "del_all_tags" {
			delAllTagsChan <- update.Message
		} else if update.Message.Command() == "stop" {
			stopMailoutChan <- update.Message
		} else if update.Message.Command() == "copy_tags" {
			copyTagsChan <- update.Message
		} else if update.Message.Command() == "get_best" {
			getBestChan <- update.Message
		} else if res, _ := regexp.MatchString(articleRegexPattern, update.Message.Text); res {
			sendIVChan <- update.Message
		} else {
			message := tgbotapi.NewMessage(update.Message.Chat.ID, "Неверная команда. Для справки введите /help")
			message.ReplyToMessageID = update.Message.MessageID
			bot.send(message)
		}
	}
}


// Notify отправляет пользователям сообщение, полученное через сайт
func (bot *HabrahabrBot) Notify(sMessage string) {
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
	}
}


// send отправляет сообщение
func (bot *HabrahabrBot) send(msg tgbotapi.MessageConfig) (tgbotapi.Message, error) {
	// TODO change?
	return bot.botAPI.Send(msg)
}


// start отвечает на команду /start hello
func (bot *HabrahabrBot) start(data chan *tgbotapi.Message) {
	for msg := range data {
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
		// Обновление переменной is_stop
		_, err = tx.Exec(`UPDATE users SET is_stop=0 WHERE id=?`, msg.Chat.ID)
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
func (bot *HabrahabrBot) help(data chan *tgbotapi.Message) {
	for msg := range data {
		message := tgbotapi.NewMessage(msg.Chat.ID, helpText)
		message.ParseMode = "HTML"
		_, err := bot.send(message)
		if err != nil {
			logging.LogSendingError(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID})
		}
	}
}


// getTags возвращает теги пользователя
func (bot *HabrahabrBot) getTags(data chan *tgbotapi.Message) {
	for msg := range data {
		rows, err := bot.db.Query(`SELECT tags FROM users WHERE id=?`, msg.Chat.ID)
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
func (bot *HabrahabrBot) addTags(data chan *tgbotapi.Message) {
	for msg := range data {
		if msg.CommandArguments() == "" {
			logging.SendErrorToUser("список тегов не может быть пустым", bot.botAPI, msg.Chat.ID)
			continue
		}

		rows, err := bot.db.Query(`SELECT tags FROM users WHERE id=?`, msg.Chat.ID)
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

		strUserTags := ""
		for k := range userTags {
			strUserTags += k + " "
		}
		strUserTags = strings.TrimSuffix(strUserTags, " ")
		tr, err := bot.db.Begin()
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		_, err = tr.Exec(`UPDATE users SET tags=? WHERE id=?`, strUserTags, msg.Chat.ID)
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		tr.Commit()

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
func (bot *HabrahabrBot) delTags(data chan *tgbotapi.Message) {
	for msg := range data {
		if msg.CommandArguments() == "" {
			logging.SendErrorToUser("список тегов не может быть пустым", bot.botAPI, msg.Chat.ID)
			continue
		}

		rows, err := bot.db.Query(`SELECT tags FROM users WHERE id=?`, msg.Chat.ID)
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

		var strUserTags string
		for k := range userTags {
			strUserTags += k + " "
		}
		strUserTags = strings.TrimSuffix(strUserTags, " ")

		tx, err := bot.db.Begin()
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		_, err = tx.Exec(`UPDATE users SET tags=? WHERE id=?`, strUserTags, msg.Chat.ID)
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
func (bot *HabrahabrBot) delAllTags(data chan *tgbotapi.Message) {
	for msg := range data {
		tx, err := bot.db.Begin()
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}
		_, err = tx.Exec(`UPDATE users SET tags='' WHERE id=?`, msg.Chat.ID)
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
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
func (bot *HabrahabrBot) copyTags(data chan *tgbotapi.Message) {
	for msg := range data {
		// Проверка ссылки, которую отправил пользователь
		userURL := msg.CommandArguments()
		res, _ := regexp.MatchString(userRegexPattern, userURL)
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

		_, err = tx.Exec(`UPDATE users SET tags=? WHERE id=?`, strUserTags, msg.Chat.ID)
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
func (bot *HabrahabrBot) stopMailoutForUser(data chan *tgbotapi.Message) {
	for msg := range data {
		tx, err := bot.db.Begin()
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}

		_, err = tx.Exec(`UPDATE users SET is_stop=1 WHERE id=?`, msg.Chat.ID)
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
func (bot *HabrahabrBot) sendIV(data chan *tgbotapi.Message, config ConfigData) {
	regexpPattern, _ := regexp.Compile(articleRegexPattern)
	for msg := range data {
		// Если сообщение попало сюда, значит, ссылка точно есть
		link := regexpPattern.FindString(msg.Text)

		instantViewURL := formatString(instantViewURL, map[string]string{"url": link})
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
func (bot * HabrahabrBot) getBest(data chan *tgbotapi.Message, config ConfigData) {
	parser := gofeed.NewParser()
	const link string = "<a href='{link}'>{title}</a>"

	for msg := range data {
		feed, err := parser.ParseURL(bestArticlesURL)
		if err != nil {
			logging.LogErrorAndNotify(logging.ErrorData{err, msg.Chat.UserName, msg.Chat.ID}, bot.botAPI)
			continue
		}	
		bestArticles := "<b>Лучшие статьи за этот день:</b>\n"
		limit := 5
		// Проверка, было ли задано другое количество статей
		if msg.CommandArguments() != "" {
			temp, err := strconv.Atoi(msg.CommandArguments())
			if err == nil && temp > 0{
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
func (bot *HabrahabrBot) mailout(config ConfigData) {
	// Parser
	parser := gofeed.NewParser()
	var lastTime time.Time
	
	// Чтение LastTime
	raw, err := ioutil.ReadFile("./data/lastArticleTime.txt")
	if err != nil {
		logging.LogFatalError("Mailout", err)
	}
	var intLastTime int64
	fmt.Sscanf(string(raw), "%d", &intLastTime)
	lastTime = time.Unix(intLastTime, 0)
	
	// Таймер
	ticker := time.NewTicker(time.Duration(config.Delay))
	var counterSendedArticles int // Подсчитывает количество отправленных статей за 1 цикл рассылки. Если превышает 25, то бот засыпает на 1 с
	
	// Первый раз статьи отправляются сразу
	for ; true; <- ticker.C {
		logging.LogEvent("Рассылка")
		counterSendedArticles = 0

		// Получение RSS-ленты
		feed, err := parser.ParseURL(allArticlesURL)
		if err != nil {
			logging.LogMinorError("Mailout", err)
			continue
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
			if lastTime.Unix() < articleTime.Unix() {
				// Создание списка тегов статьи
				var tags []string
				for _, tag := range item.Categories {
					// Форматирование от "Some Tag" к "some_tag"
					tag = strings.Replace(tag, " ", "_", -1)
					tag = strings.ToLower(tag)
					tags = append(tags, tag)
				}
				instantView := formatString(instantViewURL, map[string]string{"url": item.Link})
				message := formatString(messageText, map[string]string{"title": item.Title, "IV": instantView, "link": item.Link})

				temp := article{title: item.Title, tags: tags, link: item.Link, message: message}
				
				newArticles = append(newArticles, temp)
			} else {
				break
			}
		}
		
		// Если новых статей не было, то отправлять статьи и обновлять время не нужно
		if len(newArticles) == 0 {
			continue
		}

		// Отправка статей
		// Получение списка пользователей, которым можно отправлять статьи
		users, err := bot.db.Query(`SELECT id, tags FROM users WHERE is_stop=0`)
		if err != nil {
			logging.LogMinorError("Mailout", err)
			continue
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
					logging.LogEvent("Sleep")
					time.Sleep(time.Second)
				}
			}
		}

		// Обновление времени последней статьи (если произошла ошибка, время не меняется)
		tempTime, err := time.Parse(time.RFC1123, feed.Items[0].Published)
		if err != nil {
			logging.LogMinorError("Mailout", err)
		} else {
			lastTime = tempTime
			err = ioutil.WriteFile("./data/lastArticleTime.txt", []byte(strconv.FormatInt(lastTime.Unix(), 10)), 0644)
			if err != nil {
				logging.LogFatalError("Mailout", err)
			}
		}
	}
}