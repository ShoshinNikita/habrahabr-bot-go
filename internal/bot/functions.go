package bot

import (
	"errors"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"gopkg.in/telegram-bot-api.v4"

	"github.com/ShoshinNikita/habrahabr-bot-go/internal/logging"
)

// toSet уничтожает одинаковые элементы из массива
// Также убирает пустую строку
func toSet(slice []string) []string {
	m := make(map[string]bool)
	for _, s := range slice {
		m[s] = true
	}
	delete(m, "")

	result := make([]string, 0)
	for k := range m {
		result = append(result, k)
	}
	return result
}

// FormatString форматирует строку, подставляя вместо {name} value из args, где key == name.
func formatString(s string, args map[string]string) string {
	for key, value := range args {
		pattern := "{" + key + "}"
		s = strings.Replace(s, pattern, value, -1)
	}

	return s
}

// getRSS возвращает gofeed.Feed
// Если количество неудачных попыток получить RSS-ленту превысило лимит, то возвращается ошибка
func getRSS(source string) (*gofeed.Feed, error) {
	parser := gofeed.NewParser()
	var err error
	var feed *gofeed.Feed

	// количество попыток получить RSS-ленту
	const limit = 10
	i := 0
	for ; i < limit; i++ {
		feed, err = parser.ParseURL(source)
		if err == nil {
			break
		}
		time.Sleep(time.Millisecond * 500)
	}

	if i == limit {
		return feed, errors.New("Превышен лимит попыток получения RSS-ленты")
	}
	return feed, nil
}

// getCurrentTime возвращает текущее время
func getCurrentTime() string {
	return time.Now().Format("02.01.2006 15:04:05")
}

// logErrorAndNotify логгирует ошибку и отправляет пользователю информацию об ощибке
func (bot *Bot) logErrorAndNotify(data logging.ErrorData) {
	go logging.LogError(data)

	// Отправление сообщения об ошибке
	text := "Что-то пошло не так. Время: " + getCurrentTime() + "\nОб ошибках писать @Tirsias"
	message := tgbotapi.NewMessage(data.UserID, text)
	bot.messages <- message
}

// SendErrorToUser отправляет пользователю сообщение об ошибке (некорректный формат данных, отправленный пользователем)
func (bot *Bot) sendErrorToUser(text string, userID int64) {
	message := tgbotapi.NewMessage(userID, "Ошибка: "+text)
	bot.messages <- message
}
