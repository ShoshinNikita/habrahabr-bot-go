package bot

import (
	"strings"
	"errors"
)


// GetTagsFromString обрабатывает строку и возвращает теги в виде словаря (таким образом все теги уникальны)
func getTagsFromString(sTags string) map[string]bool {
	result := make(map[string]bool)
	if sTags != "" {
		tags := strings.Split(sTags, " ")
		for _, tag := range tags {
			result[tag] = true
		}
	}
	return result
}


// FormatString форматирует строку, подставляя вместо {name} value из args, где key == name.
// Если какого либо арумента нет, возвращается ошибка.
func formatString(s string, args map[string]string) (string, error) {
	for key, value := range args {
		pattern := "{" + key + "}"
		if strings.Index(s, pattern) == -1 {
			err := errors.New("No such argument '" + key + "'")
			return "", err
		}
		s = strings.Replace(s, pattern, value, -1)
	}

	return s, nil
}