package articlesdb

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
)

/*
*	Структура базы данных
*
*	"articles"
*		|-> id (uint64 -> string -> []byte)
*			| text
*			| time (UNIX) (int64 -> []byte)
*
*/

var dbAdapter *bolt.DB

// Open открывает базу данных
func Open(relativePath string) error {
	var err error
	dbAdapter, err = bolt.Open(relativePath, 0644, nil)
	if err != nil {
		return err
	}
	err = dbAdapter.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte("articles"))
		return err
	})

	return err
}


// Close закрывает базу данных
func Close() error {
	return dbAdapter.Close()
}


// Add добавляет текст сообщения в базу и возвращает ключ этой записи
func Add(text string, time int64) (string, error) {
	var key string
	err := dbAdapter.Update(func(tx *bolt.Tx) error {
		articles := tx.Bucket([]byte("articles"))
		id, _ := articles.NextSequence()

		// Key сначала переводим uint64 в string
		// Потом переводим в []byte
		// Позволяет легко использовать ключ при создании inline-кнопки
		key = strconv.FormatUint(id, 10)
		article, _ := articles.CreateBucket([]byte(key))
		article.Put([]byte("text"), []byte(text))

		// time переводим сразу в []byte
		// Позволяет легче сравнивать время
		bTime := make([]byte, 8)
		binary.BigEndian.PutUint64(bTime, uint64(time))
		article.Put([]byte("time"), bTime)

		return nil
	})

	return key, err
}


// Get возвращает текст сообщения
func Get(key string) (string, error) {
	var text string
	byteKey := []byte(key)

	err := dbAdapter.View(func(tx *bolt.Tx) error {
		articles := tx.Bucket([]byte("articles"))
		c := articles.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if bytes.Equal(k, byteKey) {
				article := articles.Bucket(k)
				text = string(article.Get([]byte("text")))
			}
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return text, nil
}


// CountArticles возвращает количесвто статей в базе данных
func CountArticles() int64 {
	var counter int64
	dbAdapter.View(func(tx *bolt.Tx) error {
		articles := tx.Bucket([]byte("articles"))
		c := articles.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			counter++
		}
		return nil
	})

	return counter
}


// ClearBefore удаляет все статьи, старше 7 дней
// TODO: Нужно проверить в работе
func ClearBefore() error {
	now := time.Now()
	before := now.AddDate(0, 0, -7).Unix()

	err := dbAdapter.Update(func(tx *bolt.Tx) error {
		articles := tx.Bucket([]byte("articles"))
		c := articles.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			article := articles.Bucket(k)
			bTime := article.Get([]byte("time"))
			time := int64(binary.BigEndian.Uint64(bTime))
			if time < before {
				articles.DeleteBucket(k)
			}
		}

		return nil
	})

	return err
}