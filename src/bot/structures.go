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
	ID string // ID из базы данных статей
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

// Reminder содержит информацию о напоминании
type Reminder struct {
	UserID		int64 	`json:"userID"`
	Text		string 	`json:"text"`
	Time		int64  	`json:"time"`
}