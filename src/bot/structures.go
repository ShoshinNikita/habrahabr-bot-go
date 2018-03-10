package bot

// ConfigData содержит конфигурационную информацию
type ConfigData struct {
	RssAllURL string `json:"all"`
	RssBestURL string `json:"best"`
	Token string `json:"token"`
	Delay int64	`json:"delay"`// в наносекундах
	InstantViewURL string `json:"instantView"` // ссылка на InstantView с {url} вместо ссылки на статью
	SitePassword string `json:"password"` // пароль от сайта
}

// article содержит информацию о статье
type article struct {
	title string
	tags  []string
	link  string
}