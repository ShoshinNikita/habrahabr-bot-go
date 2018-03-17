package website

import (
	"fmt"
	"html/template"
	"net/http"

	botPackage "bot"
)

var bot *botPackage.Bot
var configPassword string

// sendMessage отправляет сообщение через бота
func sendMessage(r http.Request) {
	r.ParseForm()
	message := r.Form.Get("message")
	password := r.Form.Get("password")
	if password == configPassword {
		bot.Notify(message)
	}
}

// index обрабатывает запросы на '/'
func index(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		sendMessage(*r)
	}
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		fmt.Fprint(w, "Error! Ошибка загрузки шаблона")
		return
	}
	t.Execute(w, nil)
}

// RunSite запускает сайт
func RunSite(mainBot *botPackage.Bot, password string) {
	bot = mainBot
	configPassword = password

	http.HandleFunc("/", index)
	http.ListenAndServe(":8080", nil)
}
