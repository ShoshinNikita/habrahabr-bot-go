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


//Создает клавиатуру.
func habrKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return createKeyboard([]string{"/habr_add_tags", "/habr_remove_tags"}, []string{"/habr_best"},
							[]string{"/geek_keyboard"})
}


func geekKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return createKeyboard([]string{"/geek_add_tags", "/geek_remove_tags"}, []string{"/geek_best"},
		[]string{"/habr_keyboard"})
}



//Создает клавиатуру из стрингов.
func createKeyboard(input ...[]string) tgbotapi.ReplyKeyboardMarkup {
	 parsedData := make([][]tgbotapi.KeyboardButton, 0)

	for _, array := range input {
		parsedData = append(parsedData, make([]tgbotapi.KeyboardButton, 0))

		for _, val := range array {
			parsedData[len(parsedData) - 1] = append(parsedData[len(parsedData) - 1], tgbotapi.NewKeyboardButton(val))
		}

	}

	return tgbotapi.ReplyKeyboardMarkup{
		ResizeKeyboard  : true,
		OneTimeKeyboard : false,
		Keyboard        : parsedData,
	}
}