package bot

// Константы для определения сайта
const habrArticleRegexPattern = `(https://)?(habrahabr\.ru|habr\.com|habr\.ru)/(ru|en|)/(post|company/[\w-_ ]+/blog)/\d{1,7}/?`

const habrUserRegexPattern = `^(https://)?(habrahabr\.ru|habr\.com|habr\.ru)/users/[\w\s-]+/?$`

// Текст для статьи. Нужно отформатировать функцией formatString(...)
const messageText = `{title} <a href='{IV}'>(IV)</a>

<a href='{link}'>Перейти к статье</a>

<a href='{link}#comments'>Перейти к комментариям</a>`

// ссылка на InstantView с {url} вместо ссылки на статью
const habrInstantViewURL = "https://t.me/iv?url={url}&rhash=640326b822845b"

const (
	allRuHabrArticlesURL = "https://habr.com/ru/rss/all/"
	allEnHabrArticlesURL = "https://habr.com/en/rss/all/"

	bestRuHabrArticlesURL = "https://habr.com/ru/rss/best/"
	bestEnHabrArticlesURL = "https://habr.com/en/rss/best/"
)

const helpText = `📝 <b>КОМАНДЫ</b>:
* /help – показать помощь
* /tags – показать 📃 список тегов, на которые пользователь подписан
* /add_tags – добавить теги (пример: /add_tags IT Алгоритмы)
* /del_tags – удалить теги (пример: /del_tags IT Алгоритмы)
* /del_all_tags – ❌ удалить ВСЕ теги
* /copy_tags – ✂️ скопировать теги из профиля на habrahabr'e (пример: /copy_tags https://habrahabr.ru/users/kirtis/)
* /best – получить лучшие статьи за день (по-умолчанию присылается 5, но можно через пробел указать другое количество)
* /stop – 🔕 приостановить рассылку (для продолжения рассылки - /start)

<a href= 'http://telegra.ph/Kak-polzovatsya-unofficial-habr-bot-03-09'>Дополнительная информация</a>`

/*
Команды для BotFather:

help - показать помощь
tags - показать список тегов
add_tags - добавить теги
del_tags - удалить теги
del_all_tags - удалить ВСЕ теги
copy_tags - скопировать теги из профиля на habrahabr'e
stop - приостановить рассылку
best - получить лучшие статьи за день
*/
