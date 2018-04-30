package articlesdb

import (
	"encoding/binary"
	"log"

	"github.com/boltdb/bolt"
)


// ClearAll очищает базу данных
func ClearAll() error {
	return dbAdapter.Update(func (tx *bolt.Tx) error {
		articles := tx.Bucket([]byte("articles"))
		c := articles.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			articles.DeleteBucket(k)
		}

		return nil
	})
}

// ShowAllArticles
func ShowAllArticles() error {
	return dbAdapter.View(func (tx *bolt.Tx) error {
		articles := tx.Bucket([]byte("articles"))
		c := articles.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			article := articles.Bucket(k)
			bTime := article.Get([]byte("time"))
			time := binary.BigEndian.Uint64(bTime)
			log.Print(time, "")
		}

		return nil
	}) 
}