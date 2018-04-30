package bot

import (
	"errors"
	"encoding/json"
	"io/ioutil"
	"time"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/telegram-bot-api.v4"

	artdb "articlesdb"
	"config"
	"logging"
	"userdb"
)


// mailoutBestArticles рассылает список лучших статей с Habrahabr и Geektimes пользователям,
// у которых активна рассылка статей с соответствующего ресурса
func (bot *Bot) mailoutBestArticles() {
	logging.LogEvent("Рассылка лучших статей")
	startTime := time.Now()

	users, err := userdb.GetAllUsers()
	if err != nil {
		logging.LogMinorError("mailoutBestArticles", "попытка получить список пользователей", err)
		return
	}
	var habrBestArticles, geekBestArticles string


	// Создание списка лучших статей с Habrahabr
	feed, err := getRSS(bestHabrArticlesURL)
	if err != nil {
		logging.LogMinorError("mailoutBestArticles", "попытка получить RSS-ленту Habrahabr", err)
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
	feed, err = getRSS(bestGeekArticlesURL)
	if err != nil {
		logging.LogMinorError("mailoutBestArticles", "попытка получить RSS-ленту Geektimes", err)
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

	if habrBestArticles == "" && geekBestArticles == "" {
		logging.LogMinorError("mailoutBestArticles", "списки лучших статей пусты ", errors.New(""))
		return
	}

	// Проход по всем пользователям
	var wg sync.WaitGroup
	for _, user := range users {
		wg.Add(1)
		go func(user userdb.User) {
			defer wg.Done()

			if user.HabrMailout && habrBestArticles != "" {
				message := tgbotapi.NewMessage(user.ID, habrBestArticles)
				message.ParseMode = "HTML"
				message.DisableWebPagePreview = true
				bot.send(message)
			}
			if user.GeekMailout && geekBestArticles != "" {
				message := tgbotapi.NewMessage(user.ID, geekBestArticles)
				message.ParseMode = "HTML"
				message.DisableWebPagePreview = true
				bot.send(message)
			}
		}(user)
	}

	wg.Wait()
	logging.LogEvent("Рассылка лучших статей завершена. Время выполнения: " + time.Since(startTime).String())
}


// mailout рассылает статьи с периодичностью config.Delay наносекунд
func (bot *Bot) mailout(lastTime *LastArticlesTime) {
	allUsers, err := userdb.GetAllUsers()
	if err != nil {
		logging.LogMinorError("mailout", "ошибка при попытке получить список всех пользователей", err)
		return
	}

	// Создание списка пользователей, которым нужно отправлять статьи
	var habrUsers, geekUsers []userdb.User
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
	err = habrMailout(bot, habrUsers, lastTime)
	if err != nil {
		logging.LogMinorError("habrMailout", "вызов habrMailout", err)
	}
	logging.LogEvent("Завершена. Время выполнения: " + time.Since(startTime).String())

	time.Sleep(time.Second * 1)

	// Рассылка статей с Geektimes
	logging.LogEvent("Рассылка статей с Geektimes")
	startTime = time.Now()
	err = geekMailout(bot, geekUsers, lastTime)
	if err != nil {
		logging.LogMinorError("geekMailout", "вызов geekMailout", err)
	}
	logging.LogEvent("Завершена. Время выполнения: " + time.Since(startTime).String())

	// Перезапись времени
	raw, _ := json.Marshal(lastTime)
	err = ioutil.WriteFile(config.Data.Prefix + "data/lastArticleTime.json", raw, 0644)
	if err != nil {
		logging.LogFatalError("Mailout", "попытка записать файл lastArticleTime.json", err)
	}
}


// createInlineButton создаёт новую inline-клавиатуру, добавляя к команде "remind" id сообщения
func createInlineButton(id string) tgbotapi.InlineKeyboardMarkup {
	sixHoursButton := tgbotapi.NewInlineKeyboardButtonData("Напомнить через 6 часов", "remind"+id)
	return tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(sixHoursButton))
}


// habrMailout отвечает за рассылку статей с сайта Habrahabr.ru
func habrMailout(bot *Bot, allUsers []userdb.User, lastTime *LastArticlesTime) error {
	// Получение RSS-ленты
	feed, err := getRSS(allHabrArticlesURL)
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
			// добавление ID из базы данных
			temp.ID, _ = artdb.Add(temp.message, time.Now().Unix())
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
	var wg sync.WaitGroup
	for _, user := range allUsers {
		wg.Add(1)

		// Проход по всем статьям в обратном порядке
		go func(user userdb.User){
			defer wg.Done()

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
							}
						}
					}
				}

				// Отправка пользователю
				if shouldSend {
					message := tgbotapi.NewMessage(user.ID, newArticles[i].message)
					message.ParseMode = "HTML"
					message.ReplyMarkup = createInlineButton(newArticles[i].ID)
					bot.send(message)
				}
			}
		}(user)
	}
	wg.Wait()

	// Обновление времени
	tempTime, err := time.Parse(time.RFC1123, feed.Items[0].Published)
	if err != nil {
		return err
	}
	lastTime.Habr = tempTime.Unix()

	return nil
}


// geekMailout отвечает за рассылку статей с сайта Geektimes.ru
func geekMailout(bot *Bot, allUsers []userdb.User, lastTime *LastArticlesTime) error {
	// Получение RSS-ленты
	feed, err := getRSS(allGeekArticlesURL)
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
			// добавление ID из базы данных
			temp.ID, _ = artdb.Add(temp.message, time.Now().Unix())
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
	var wg sync.WaitGroup
	for _, user := range allUsers {
		wg.Add(1)

		go func(user userdb.User) {
			defer wg.Done()

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
							}
						}
					}
				}

				// Отправка пользователю
				if shouldSend {
					message := tgbotapi.NewMessage(user.ID, newArticles[i].message)
					message.ParseMode = "HTML"
					message.ReplyMarkup = createInlineButton(newArticles[i].ID)
					bot.send(message)
				}
			}
		}(user)
	}
	wg.Wait()

	// Обновление времени
	tempTime, err := time.Parse(time.RFC1123, feed.Items[0].Published)
	if err != nil {
		return err
	}
	lastTime.Geek = tempTime.Unix()

	return nil
}