# Habrahabr-бот на Go #

Неофициальный бот для рассылки статей с сайта [habrahabr.ru](https://habrahabr.ru/top/) в Telegram

## Требования ##

* Язык - go1.10
* Сторонние библиотеки:
	* Telegram Bot API – [telegram-bot-api.v4](http://gopkg.in/telegram-bot-api.v4)
	* SQLite3 driver – [go-sqlite3](https://github.com/mattn/go-sqlite3)
	* RSS парсер – [gofeed](https://github.com/mmcdole/gofeed)
	* Web scraper – [soup](https://github.com/anaskhan96/soup)
	* Парсер дат и времени – [jodaTime](https://github.com/vjeantet/jodaTime)

## Информация о работе ##

Бот использует [RSS-ленту](https://habrahabr.ru/rss/all) сайта [habrahabr.ru](https://habrahabr.ru/top/) для получения списка статей. Данные пользователей (id, теги) хранятся в SQLite базе данных.

## Файлы и их описание ##

### Структура папок исходного кода ###

* bot
	* bot.go – основная часть, отвечающая за бота
	* functions.go – полезные функции
	* structures.go – структуры, которые используются в боте
	* constants.go - константы
* logging
	* logging.go – отвечает за логгирование всего, что происходит в программе
* main
	* data
		* config.json
		* database.db
		* lastArticleTime.txt
	* logs
		* Log-файлы (создаются автоматически)
	* templates
		* index.html - страница отправки сообщений
	* main.go – главный файл
* website
	* website.go – основная часть, отвечающая за сайт

### Содержание файлов ###

* Файл config.json содержит конфигурационную информацию:

```json
{
	"token": "***BOT TOKEN***",
	"delay": "***TIME IN NANOSECONDS***",
	"password": "***PASSWORD FOR WEBSITE***"
}
```

* Файл database.db – SQLite база данных.

```sql
CREATE TABLE `users` (
	`id`	INTEGER,
	`tags`	TEXT DEFAULT "",
	`is_stop`	INTEGER DEFAULT 0,
	PRIMARY KEY(`id`)
);
```

* Файл lastArticleTime.txt содержит одну строку – время в UNIX формате (например, 1520337177)

## Лицензия ##

[MIT License](LICENSE)