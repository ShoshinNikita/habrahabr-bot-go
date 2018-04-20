package bot

import (
	"strings"
	"gopkg.in/telegram-bot-api.v4"
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