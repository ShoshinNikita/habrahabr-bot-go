package bot

const habrArticleRegexPattern = "(?:https://|)habrahabr.ru/(?:post|company/\\w+/blog)/\\d{1,6}(?:/|)"
const geekArticleRegexPattern = "(?:https://|)geektimes.ru/(?:post|company/\\w+/blog)/\\d{1,6}(?:/|)"

const habrUserRegexPattern = "^https://habrahabr.ru/users/[\\w\\s_]+/$"
const geekUserRegexPattern = "^https://geektimes.ru/users/[\\w\\s_]+/$"

const messageText = "[{source}] {title} <a href='{IV}'>(IV)</a>\n\n<a href='{link}'>Перейти к статье</a>\n\n<a href='{link}#comments'>Перейти к комментариям</a>"

const maxArticlesLimit = 25 // Служит для ограничения отправки статей, чтобы Telegram не заблокировал бота

// ссылка на InstantView с {url} вместо ссылки на статью
const habrInstantViewURL = "https://t.me/iv?url={url}&rhash=2cb77307aed3b2"
const geekInstantViewURL = "https://t.me/iv?url={url}&rhash=267de544beb71f"

const allHabrArticlesURL = "https://habrahabr.ru/rss/all/?with_hubs=true:?with_tags=true:"
const bestHabrArticlesURL = "https://habrahabr.ru/rss/best/?with_hubs=true:?with_tags=true:"

const allGeekArticlesURL = "https://geektimes.ru/rss/all/?with_hubs=true:?with_tags=true:"
const bestGeekArticlesURL = "https://geektimes.ru/rss/best/?with_hubs=true:?with_tags=true:"

// Константы для определения сайта
const geek = "geektimes"
const habr = "habrahabr"

const helpText = `
📝 <b>КОМАНДЫ</b>:
* /help – показать помощь
* /habr_tags (/geek_tags) – показать 📃 список тегов, на которые пользователь подписан
* /habr_add_tags (/geek_add_tags) – добавить теги (пример: /habr_add_tags IT Алгоритмы)
* /habr_del_tags (/geek_del_tags) – удалить теги (пример: /habr_del_tags IT Алгоритмы)
* /habr_del_all_tags (/geek_del_all_tags) – ❌ удалить ВСЕ теги
* /habr_copy_tags (/geek_copy_tags) – ✂️ скопировать теги из профиля на habrahabr'e (пример: /habr_copy_tags https://habrahabr.ru/users/kirtis/)
* /habr_best (/geek_best) – получить лучшие статьи за день (по-умолчанию присылается 5, но можно через пробел указать другое количество)
* /habr_stop (/geek_stop) – 🔕 приостановить рассылку (для продолжения рассылки - /start)

<a href= 'http://telegra.ph/Kak-polzovatsya-unofficial-habr-bot-03-09'>Дополнительная информация</a>
`

const botFatherCommands = `
help - показать помощь
habr_tags - показать список тегов
habr_add_tags - добавить теги
habr_del_tags - удалить теги
habr_del_all_tags - удалить ВСЕ теги
habr_copy_tags - скопировать теги из профиля на habrahabr'e
habr_stop - приостановить рассылку
habr_best - получить лучшие статьи за день
geek_tags - показать список тегов
geek_add_tags - добавить теги
geek_del_tags - удалить теги
geek_del_all_tags - удалить ВСЕ теги
geek_copy_tags - скопировать теги из профиля на geektimes'e
geek_stop - приостановить рассылку
geek_best - получить лучшие статьи за день
`
