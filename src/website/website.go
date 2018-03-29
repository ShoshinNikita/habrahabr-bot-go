package website

import (
	"fmt"
	"html/template"
	"net/http"

	botPackage "bot"
	"config"
)

var bot *botPackage.Bot


// sendMessage отправляет сообщение через бота
func sendMessage(r http.Request) {
	r.ParseForm()
	message := r.Form.Get("message")
	password := r.Form.Get("password")
	if password == config.Data.SitePassword {
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
func RunSite(mainBot *botPackage.Bot) {
	bot = mainBot

	http.HandleFunc("/", index)
	http.ListenAndServe(":8080", nil)
}
