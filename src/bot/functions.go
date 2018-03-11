package bot

import (
	"strings"
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
func formatString(s string, args map[string]string) (string) {
	for key, value := range args {
		pattern := "{" + key + "}"
		s = strings.Replace(s, pattern, value, -1)
	}
	
	return s
}