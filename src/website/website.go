package website

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
	
	"github.com/gorilla/mux"

	botPackage "bot"
	"config"
	"db"
)

var bot *botPackage.Bot


// index обрабатывает запросы на '/'
func index(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie("authorized"); err != nil {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}

	t, err := template.ParseFiles(config.Data.Prefix + "templates/index.html")
	if err != nil {
		fmt.Fprint(w, "Error! Ошибка загрузки шаблона")
		return
	}
	data := make(map[string]interface{})
	data["usersNumber"] = db.GetUsersNumber()
	t.Execute(w, data)
}

// auth обрабатывает запросы на '/auth'
func auth(w http.ResponseWriter, r *http.Request) {
	// Есть куки
	if _, err := r.Cookie("authorized"); err == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	t, err := template.ParseFiles(config.Data.Prefix + "templates/auth.html")
	if err != nil {
		fmt.Fprint(w, "Error! Ошибка загрузки шаблона")
		return
	}

	t.Execute(w, nil)
}


// returnUser возвращает данные о пользователе в виде json
func returnUser(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie("authorized"); err != nil {
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	user, err := db.GetUser(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(user)
}


// checkPass проверяет пароль
func checkPass(w http.ResponseWriter, r *http.Request) {	
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.Form.Get("password") == config.Data.SitePassword {
		cookie := http.Cookie{Name: "authorized", Expires: time.Now().Add(30 * 24 * time.Hour)}
		http.SetCookie(w, &cookie)

		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
	}
}


// send отправляет сообщение пользователям
func send(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if password := r.Form.Get("password"); password == config.Data.SitePassword {
		message := r.Form.Get("message")
		bot.Notify(message)
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}


// RunSite запускает сайт
func RunSite(mainBot *botPackage.Bot) {
	bot = mainBot
	
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("GET").Path("/").HandlerFunc(index)
	router.Methods("GET").Path("/auth").HandlerFunc(auth)
	router.Methods("GET").Path("/user").HandlerFunc(returnUser)
	router.Methods("POST").Path("/auth").HandlerFunc(checkPass)
	router.Methods("POST").Path("/send").HandlerFunc(send)

	http.ListenAndServe(":8080", router)
}
