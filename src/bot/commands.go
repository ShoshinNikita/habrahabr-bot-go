package bot

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/anaskhan96/soup"   // html parser
	"github.com/mmcdole/gofeed"    // Rss parser
	"gopkg.in/telegram-bot-api.v4" // Telegram api

	"userdb"	// взаимодействие с базой данных
	"logging" 	// логгирование
)


// start отвечает на команду /start, создаёт запись о пользователе
func (bot *Bot) start(msg *tgbotapi.Message) {
	// Создание пользователя
	err := userdb.CreateUser(strconv.FormatInt(msg.Chat.ID, 10))
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
		err = userdb.StartMailout(strconv.FormatInt(msg.Chat.ID, 10), habr)
	} else if site == geek {
		err = userdb.StartMailout(strconv.FormatInt(msg.Chat.ID, 10), geek)
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
		err = userdb.StopMailout(strconv.FormatInt(msg.Chat.ID, 10), habr)
	} else if site == geek {
		err = userdb.StopMailout(strconv.FormatInt(msg.Chat.ID, 10), geek)
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

	user, err := userdb.GetUser(strconv.FormatInt(msg.Chat.ID, 10))
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
		updatedTags, err = userdb.AddUserTags(strconv.FormatInt(msg.Chat.ID, 10), habr, newTags)
	} else if site == geek {
		updatedTags, err = userdb.AddUserTags(strconv.FormatInt(msg.Chat.ID, 10), geek, newTags)
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
		updatedTags, err = userdb.DelUserTags(strconv.FormatInt(msg.Chat.ID, 10), habr, tagsForDel)
	} else if site == geek {
		updatedTags, err = userdb.DelUserTags(strconv.FormatInt(msg.Chat.ID, 10), geek, tagsForDel)
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
		err = userdb.DelAllUserTags(strconv.FormatInt(msg.Chat.ID, 10), habr)
	} else if site == geek {
		err = userdb.DelAllUserTags(strconv.FormatInt(msg.Chat.ID, 10), habr)
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
			err = userdb.UpdateTags(strconv.FormatInt(msg.Chat.ID, 10), habr, userTags)
		}
		case geek: {
			err = userdb.UpdateTags(strconv.FormatInt(msg.Chat.ID, 10), habr, userTags)
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