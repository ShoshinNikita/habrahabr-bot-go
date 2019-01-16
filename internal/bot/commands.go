package bot

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/anaskhan96/soup"   // html parser
	"github.com/mmcdole/gofeed"    // Rss parser
	"gopkg.in/telegram-bot-api.v4" // Telegram api

	"github.com/ShoshinNikita/habrahabr-bot-go/internal/logging" // логгирование
	"github.com/ShoshinNikita/habrahabr-bot-go/internal/userdb"  // взаимодействие с базой данных
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
		bot.logErrorAndNotify(data)
		return
	}

	message := tgbotapi.NewMessage(msg.Chat.ID, "Привет, "+msg.Chat.UserName+"! Введи /help для справки")
	bot.messages <- message
}

// stopMailout останавливает рассылку для пользователя
func (bot *Bot) stopMailout(msg *tgbotapi.Message) {
	err := userdb.StopMailout(strconv.FormatInt(msg.Chat.ID, 10))
	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/...stop",
			AddInfo:  "попытка остановить рассылку"}
		bot.logErrorAndNotify(data)
		return
	}

	message := tgbotapi.NewMessage(msg.Chat.ID, "Рассылка приостановлена")
	bot.messages <- message
}

// help отправляет справочную информацию
func (bot *Bot) help(msg *tgbotapi.Message) {
	message := tgbotapi.NewMessage(msg.Chat.ID, helpText)
	message.ParseMode = "HTML"
	bot.messages <- message
}

// getStatus возвращает теги пользователя и информация, осуществляется ли рассылка
func (bot *Bot) getStatus(msg *tgbotapi.Message) {
	user, err := userdb.GetUser(strconv.FormatInt(msg.Chat.ID, 10))
	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/...tags",
			AddInfo:  "попытка получить данные пользователя"}
		bot.logErrorAndNotify(data)
		return
	}

	tags := user.Tags

	var text string
	if len(tags) == 0 {
		text = "Список тегов пуст"
	} else {
		text = "Список тегов:\n* "
		text += strings.Join(tags, "\n* ")
	}

	text += "\n\n📬 Рассылка: "

	if user.Mailout {
		text += "осуществляется"
	} else {
		text += "не осуществляется"
	}

	message := tgbotapi.NewMessage(msg.Chat.ID, text)
	bot.messages <- message
}

// addTags добавляет теги, которые прислал пользователь
func (bot *Bot) addTags(msg *tgbotapi.Message) {
	newTags := strings.Split(strings.ToLower(msg.CommandArguments()), " ")
	newTags = toSet(newTags)
	if len(newTags) == 0 {
		bot.sendErrorToUser("список тегов не может быть пустым", msg.Chat.ID)
		return
	}

	updatedTags, err := userdb.AddUserTags(strconv.FormatInt(msg.Chat.ID, 10), newTags)

	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/...add_tags",
			AddInfo:  "попытка добавить теги"}
		bot.logErrorAndNotify(data)
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
	bot.messages <- message
}

// delTags удаляет теги, которые прислал пользователь
func (bot *Bot) delTags(msg *tgbotapi.Message) {
	tagsForDel := strings.Split(strings.ToLower(msg.CommandArguments()), " ")
	tagsForDel = toSet(tagsForDel)
	if len(tagsForDel) == 0 {
		bot.sendErrorToUser("список тегов не может быть пустым", msg.Chat.ID)
		return
	}

	updatedTags, err := userdb.DelUserTags(strconv.FormatInt(msg.Chat.ID, 10), tagsForDel)

	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/...del_tags",
			AddInfo:  "попытка удалить теги"}
		bot.logErrorAndNotify(data)
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
	bot.messages <- message
}

// delAllTags очищает список тегов пользователя
func (bot *Bot) delAllTags(msg *tgbotapi.Message) {
	err := userdb.DelAllUserTags(strconv.FormatInt(msg.Chat.ID, 10))

	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/...del_all_tags",
			AddInfo:  "попытка удалить теги"}
		bot.logErrorAndNotify(data)
		return
	}

	message := tgbotapi.NewMessage(msg.Chat.ID, "Список тегов очищен")
	bot.messages <- message
}

// copyTags копирует теги пользователя со страницы на Habrahabr
func (bot *Bot) copyTags(msg *tgbotapi.Message) {
	userURL := msg.CommandArguments()
	res, _ := regexp.MatchString(habrUserRegexPattern, userURL)

	// Проверка ссылки, которую отправил пользователь
	if !res {
		bot.sendErrorToUser("неверный формат ссылки", msg.Chat.ID)
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
		bot.logErrorAndNotify(data)
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
	tags = doc.FindAll("a", "class", "list-snippet__title-link")
	for _, company := range tags {
		tag := "Блог компании " + company.Text()
		tag = strings.ToLower(tag)
		tag = strings.Replace(tag, " ", "_", -1)
		userTags = append(userTags, tag)
	}

	if len(userTags) == 0 {
		bot.sendErrorToUser("было обнаружено 0 тегов. Должно быть больше", msg.Chat.ID)
		return
	}

	err = userdb.UpdateTags(strconv.FormatInt(msg.Chat.ID, 10), userTags)
	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/...copy_tags",
			AddInfo:  "попытка перезаписать теги"}
		bot.logErrorAndNotify(data)
		return
	}

	text := "Теги обновлены. Список тегов:\n* " + strings.Join(userTags, "\n* ")
	message := tgbotapi.NewMessage(msg.Chat.ID, text)
	bot.messages <- message
}

// sendIV отправляет пользователю ссылку на статью, которую он прислал, в виде InstantView
func (bot *Bot) sendIV(msg *tgbotapi.Message) {
	habrRegexpPattern := regexp.MustCompile(habrArticleRegexPattern)

	link := habrRegexpPattern.FindString(msg.Text)
	instantViewURL := formatString(habrInstantViewURL, map[string]string{"url": link})

	text := "<a href=\"" + instantViewURL + "\">InstantView</a>\n\n" +
		"<a href=\"" + link + "\">Перейти к статье</a>\n\n" +
		"<a href=\"" + link + "#comments\">Перейти к комментариям</a>"

	message := tgbotapi.NewMessage(msg.Chat.ID, text)
	message.ParseMode = "HTML"
	bot.messages <- message
}

// getBest отправляет пользователю лучшие статьи за сегодняшний день.
// По-умолчанию – 5, если пользователь указал другое число - другое
func (bot *Bot) getBest(msg *tgbotapi.Message) {
	parser := gofeed.NewParser()
	feed, err := parser.ParseURL(bestRuHabrArticlesURL)
	if err != nil {
		data := logging.ErrorData{
			Error:    err,
			Username: msg.Chat.UserName,
			UserID:   msg.Chat.ID,
			Command:  "/...best",
			AddInfo:  "попытка распарсить RSS-ленту"}
		bot.logErrorAndNotify(data)
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
	bot.messages <- message
}
