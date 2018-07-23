package bot

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	"gopkg.in/telegram-bot-api.v4"

	"github.com/ShoshinNikita/habrahabr-bot-go/internal/config"
	"github.com/ShoshinNikita/habrahabr-bot-go/internal/logging"
	"github.com/ShoshinNikita/habrahabr-bot-go/internal/userdb"
)

// smartQueue хранит список уникальных строк (url) в виде очереди
// Максимальный размер равен size. При переполнении первый элемент удаляется
type smartQueue struct {
	queue []string
	size  int
}

// newSmartQueue возвращает новый объект
func newSmartQueue(size int, items []string) smartQueue {
	q := smartQueue{}
	q.size = size
	q.queue = make([]string, len(items))
	copy(q.queue, items)
	return q
}

// add добавляет в очередь только уникальные строки (ссылки на статьи)
func (m *smartQueue) add(s string) {
	shouldAdd := true
	// Проверяем уникальность строки
	for _, q := range m.queue {
		if q == s {
			shouldAdd = false
			break
		}
	}

	if shouldAdd {
		m.queue = append(m.queue, s)
		if len(m.queue) > m.size {
			m.queue = append([]string{}, m.queue[1:]...)
		}
	}
}

// contains проверяет, содержится ли элемент в очереди
func (m *smartQueue) contains(s string) bool {
	isThere := false
	for _, q := range m.queue {
		if q == s {
			isThere = true
			break
		}
	}
	return isThere
}

// Хранят старые записи из RSS-ленты. Служат для получения всех новых статей (даже тех, которые были скрыты на время)
// Задаются в функции bot.StartPooling(). Из-за этого make(map[string]gofeed.Item) не нужен.
var oldArticles smartQueue

// getAllArticles возвращает все gofeed.Item (в порядке убывания по времени, т.е новые раньше)
func getAllArticles() ([]gofeed.Item, error) {
	// Получение RSS-ленты
	feed, err := getRSS(allHabrArticlesURL)
	if err != nil {
		return []gofeed.Item{}, err
	}

	// Создание списка новых статей
	var result []gofeed.Item
	for _, item := range feed.Items {
		result = append(result, *item)
	}

	return result, nil
}

// getNewArticles возвращает только новые статьи
// Логика работы:
// 1) Получаем все статьи из RSS-ленты
// 2) Сравниваем их со статьями из предыдущего обновления (по URL). Если URL одинаковые, то удаляем статью
// 3) Отправляем статьи в канал
func getNewArticles(newArticlesChan chan<- article) {
	ticker := time.NewTicker(time.Second * time.Duration(config.Data.Delay))
	for ; true; <-ticker.C {
		allItems, err := getAllArticles()
		if err != nil {
			logging.LogMinorError("getNewArticles", "Попытка получить статьи", err)
			continue
		}

		// Отбираем только новые статьи
		var newItems []gofeed.Item
		for _, item := range allItems {
			if !oldArticles.contains(item.Link) {
				newItems = append(newItems, item)
			}
		}

		// Проходим только по новым статьям в обратном порядке и отправляем их в канал
		for i := len(newItems) - 1; i >= 0; i-- {
			// Создание списка тегов статьи
			var tags []string
			for _, tag := range newItems[i].Categories {
				// Форматирование от "Some Tag" к "some_tag"
				tag = strings.Replace(tag, " ", "_", -1)
				tag = strings.ToLower(tag)
				tags = append(tags, tag)
			}
			// Создания текста сообщения
			instantView := formatString(habrInstantViewURL, map[string]string{"url": newItems[i].Link})
			message := formatString(messageText,
				map[string]string{
					"title": newItems[i].Title,
					"IV":    instantView,
					"link":  newItems[i].Link})

			article := article{title: newItems[i].Title, tags: tags, link: newItems[i].Link, message: message}

			newArticlesChan <- article
		}

		// Обновляем список старых статей
		for _, item := range allItems {
			oldArticles.add(item.Link)
		}
	}
}

// mailoutBestArticles рассылает список лучших статей с Habrahabr
func (bot *Bot) mailoutBestArticles() {
	logging.LogEvent("Рассылка лучших статей")
	const limit = 7

	users, err := userdb.GetAllUsers()
	if err != nil {
		logging.LogMinorError("mailoutBestArticles", "попытка получить список пользователей", err)
		return
	}
	var habrBestArticles string

	// Создание списка лучших статей с Habrahabr
	feed, err := getRSS(bestHabrArticlesURL)
	if err != nil {
		logging.LogMinorError("mailoutBestArticles", "попытка получить RSS-ленту Habrahabr", err)
	} else {
		habrBestArticles = "<b>Лучшие статьи за этот день на Habrahabr:</b>\n"

		// Создание списка статей (в виде строки)
		for i, item := range feed.Items {
			if i >= limit {
				break
			}
			number := strconv.Itoa(i + 1)
			habrBestArticles += number + ") " + formatString("<a href='{link}'>{title}</a>", map[string]string{"link": item.Link, "title": item.Title}) + "\n"
		}
	}

	if habrBestArticles == "" {
		logging.LogMinorError("mailoutBestArticles", "списки лучших статей пусты ", errors.New(""))
		return
	}

	// Проход по всем пользователям
	var wg sync.WaitGroup
	for _, user := range users {
		wg.Add(1)

		go func(user userdb.User) {
			defer wg.Done()

			if user.Mailout {
				message := tgbotapi.NewMessage(user.ID, habrBestArticles)
				message.ParseMode = "HTML"
				message.DisableWebPagePreview = true
				bot.messages <- message
			}
		}(user)
	}

	wg.Wait()
	logging.LogEvent("Рассылка лучших статей завершена.")
}

// mailout рассылает статьи с периодичностью config.Delay наносекунд
func (bot *Bot) mailout() {
	go getNewArticles(bot.articles)

	var (
		allUsers []userdb.User
		err      error
	)

	for newArticle := range bot.articles {
		allUsers, err = userdb.GetAllUsers()
		if err != nil {
			logging.LogMinorError("mailout", "ошибка при попытке получить список всех пользователей", err)
			return
		}

		// Создание списка пользователей, которым нужно отправлять статьи
		users := func() (users []userdb.User) {
			for _, user := range allUsers {
				if user.Mailout {
					users = append(users, user)
				}
			}
			return users
		}()

		var wg sync.WaitGroup
		for _, user := range users {
			wg.Add(1)
			go func(user userdb.User) {
				defer wg.Done()

				if shouldSend(user, newArticle) {
					message := tgbotapi.NewMessage(user.ID, newArticle.message)
					message.ParseMode = "HTML"
					bot.messages <- message
				}
			}(user)
		}
		wg.Wait()

		// Обновление ссылок в файле
		allArticles := struct {
			HabrArticles []string `json:"habr"`
		}{
			oldArticles.queue,
		}

		raw, _ := json.Marshal(allArticles)
		err = ioutil.WriteFile("data/lastArticles.json", raw, 0644)
		if err != nil {
			logging.LogMinorError("Mailout", "попытка записать файл lastArticles.json", err)
		}
	}
}

// habrMailout отвечает за рассылку статей с сайта Habrahabr.ru
func shouldSend(user userdb.User, newArticle article) bool {
	if len(user.Tags) == 0 {
		return true
	}

	// Проверка, есть ли теги пользователя в статье
	for _, tag := range newArticle.tags {
		for _, userTag := range user.Tags {
			if tag == userTag {
				return true
			}
		}
	}

	return false
}
