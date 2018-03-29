package bot

import (
	"gopkg.in/telegram-bot-api.v4"
)


// article содержит информацию о статье
type article struct {
	title string
	link  string
	tags  []string
	message string
}

// LastArticlesTime хранит время последних статей на Habrahabr и на Geektimes
type LastArticlesTime struct {
	Habr int64 `json:"habr"`
	Geek int64 `json:"geek"`
}

type userCommand struct {
	message *tgbotapi.Message
	site string 
}