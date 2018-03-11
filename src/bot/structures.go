package bot

// ConfigData содержит конфигурационную информацию
type ConfigData struct {
	Token string `json:"token"`
	Delay int64	`json:"delay"`// в наносекундах
	SitePassword string `json:"password"` // пароль от сайта
}

// article содержит информацию о статье
type article struct {
	title string
	link  string
	tags  []string
	message string
}