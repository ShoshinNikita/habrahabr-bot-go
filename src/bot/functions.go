package bot

import (
	"errors"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"gopkg.in/telegram-bot-api.v4"

	artdb "articlesdb"
	"logging"
)


// FormatString форматирует строку, подставляя вместо {name} value из args, где key == name.
func formatString(s string, args map[string]string) (string) {
	for key, value := range args {
		pattern := "{" + key + "}"
		s = strings.Replace(s, pattern, value, -1)
	}
	
	return s
}


// createKeyboard формирует клавиатуру
func createKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := make([][]tgbotapi.KeyboardButton, 0)

	keyboard = append(keyboard, []tgbotapi.KeyboardButton{{Text: "/habr_best"}, {Text: "/habr_tags"}})
	keyboard = append(keyboard, []tgbotapi.KeyboardButton{{Text: "/geek_best"}, {Text: "/geek_tags"}})

	return tgbotapi.ReplyKeyboardMarkup{
		Keyboard        : keyboard,
		ResizeKeyboard  : true,
		OneTimeKeyboard : false,
	}
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


// clearArticlesDB является обёрткой над artdb.ClearBefore()
func clearArticlesDB() {
	err := artdb.ClearBefore()
	if err != nil {
		logging.LogMinorError("clearArticlesDB", "ошибка при попытке очистить БД", err)
	}
}