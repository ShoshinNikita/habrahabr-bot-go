package db

import (
	"strconv"
	"strings"
	"errors"
)

func toSlice(data []byte) ([]string, error) {
	findElement := func(slice []string, key string) int {
		for i, elem := range slice {
			if elem == key {
				return i
			}
		}
		return -1
	}

	if data == nil {
		return []string{}, errors.New("Field doesn't exist")
	}
	
	sData := string(data)
	result := strings.Split(sData, " ")
	if index := findElement(result, ""); index != -1 {
		result[index], result[len(result) - 1] = result[len(result) - 1], result[index]
		result = result[:len(result) - 1]
	}
	return result, nil
}


func toBool(data []byte) (bool, error) {
	if data == nil {
		return false, errors.New("Field doesn't exist")
	}

	return strconv.ParseBool(string(data))
}


func toInt64(data []byte) (int64, error) {
	if data == nil {
		return 0, errors.New("Field doesn't exist")
	}

	return strconv.ParseInt(string(data), 10, 64)
}


func addTags(oldTags, newTags []string) []string {
	oldTagsMap := make(map[string]int)
	for _, tag := range oldTags {
		oldTagsMap[tag] = 1
	}
	
	for _, tag := range newTags {
		oldTagsMap[tag] = 1
	}
	// удаление пустой строки
	delete(oldTagsMap, "")

	result := make([]string, 0)
	for tag := range oldTagsMap {
		result = append(result, tag)
	}

	return result
}


func delTags(oldTags, tagsForDel []string) []string {
	// Создаём map
	oldTagsMap := make(map[string]int)
	for _, tag := range oldTags {
		oldTagsMap[tag] = 1
	}

	for _, tag := range tagsForDel {
		delete(oldTagsMap, tag)
	}
	delete(oldTagsMap, "")
	
	result := make([]string, 0)
	for tag := range oldTagsMap {
		result = append(result, tag)
	}

	return result
}