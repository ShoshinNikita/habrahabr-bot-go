package userdb

import (
	"errors"
	"strings"

	"github.com/boltdb/bolt"
)

/*
*	Структура базы данных
*
*	"users"
*		|-> id
*			| Tags
*			| Mailout
*
 */

// User содержит в себе информацию о пользователе
type User struct {
	ID      int64    `json:"id"`
	Tags    []string `json:"tags"`
	Mailout bool     `json:"mailout"`
}

var dbAdapter *bolt.DB

// Open открывает базу данных (или создаёт, если не существует)
func Open(relativePath string) error {
	var err error
	dbAdapter, err = bolt.Open(relativePath, 0600, nil)
	if err != nil {
		return err
	}

	err = dbAdapter.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("users"))
		return err
	})
	if err != nil {
		return err
	}

	return nil
}

// Close закрывает базу данных
func Close() {
	dbAdapter.Close()
}

// CreateUser создаёт запись пользователя. Если запись существует, то включает ему рассылку
func CreateUser(id string) error {
	err := dbAdapter.Update(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))
		userBucket := usersBucket.Bucket([]byte(id))

		// Пользователь использует бота в первый раз
		if userBucket == nil {
			userBucket, err := usersBucket.CreateBucket([]byte(id))
			if err != nil {
				return err
			}
			userBucket.Put([]byte("Tags"), []byte(""))
			userBucket.Put([]byte("Mailout"), []byte("true"))
		} else {
			// Если пользователь существовал, то просто включаем ему рассылку
			userBucket.Put([]byte("Mailout"), []byte("true"))
		}

		return nil
	})

	return err
}

// GetUser возвращает пользовательские данные
func GetUser(id string) (User, error) {
	var user User
	var err error

	err = dbAdapter.View(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))

		userBucket := usersBucket.Bucket([]byte(id))
		if userBucket == nil {
			return errors.New("User with id '" + id + "' doesn't exist")
		}

		user.ID, err = toInt64([]byte(id))
		if err != nil {
			return err
		}
		user.Tags, err = toSlice(userBucket.Get([]byte("Tags")))
		if err != nil {
			return err
		}
		user.Mailout, err = toBool(userBucket.Get([]byte("Mailout")))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return User{}, err
	}

	return user, nil
}

// GetAllUsers возвращает slice, содержащий данные о всех пользователях
func GetAllUsers() ([]User, error) {
	users := make([]User, 0)

	err := dbAdapter.View(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))

		c := usersBucket.Cursor()
		var userBucket *bolt.Bucket

		// В бакете содержатся только другие бакеты
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			userBucket = usersBucket.Bucket(k)

			var user User
			var err error

			user.ID, err = toInt64(k)
			if err != nil {
				continue
			}
			user.Tags, err = toSlice(userBucket.Get([]byte("Tags")))
			if err != nil {
				continue
			}
			user.Mailout, err = toBool(userBucket.Get([]byte("Mailout")))
			if err != nil {
				continue
			}

			users = append(users, user)

		}

		return nil
	})
	if err != nil {
		return []User{}, err
	}

	return users, nil
}

// GetUsersNumber возвращает количество пользователей
func GetUsersNumber() int64 {
	var counter int64
	dbAdapter.View(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))
		c := usersBucket.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			counter++
		}
		return nil
	})

	return counter
}

// AddUserTags добавляет теги, которые были переданы
// Возвращает slice, содержащий обновлённые теги
func AddUserTags(id string, newTags []string) ([]string, error) {
	updatedTags := make([]string, 0)

	err := dbAdapter.Update(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))

		userBucket := usersBucket.Bucket([]byte(id))
		if userBucket == nil {
			return errors.New("User with id '" + id + "' doesn't exist")
		}

		// Получение slice из старых тегов
		oldTags, err := toSlice(userBucket.Get([]byte("Tags")))
		if err != nil {
			return err
		}

		// Добавление тегов
		updatedTags = addTags(oldTags, newTags)

		updatedTagsString := strings.Join(updatedTags, " ")
		userBucket.Put([]byte("Tags"), []byte(updatedTagsString))

		return nil
	})
	if err != nil {
		return []string{}, err
	}

	return updatedTags, nil
}

// UpdateTags перезаписывает теги
func UpdateTags(id string, tags []string) error {
	err := dbAdapter.Update(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))

		userBucket := usersBucket.Bucket([]byte(id))
		if userBucket == nil {
			return errors.New("User with id '" + id + "' doesn't exist")
		}

		tagsString := strings.Join(tags, " ")

		userBucket.Put([]byte("Tags"), []byte(tagsString))

		return nil
	})

	return err
}

// DelUserTags удаляет теги, которые были переданы
// Возвращает slice, содержащий обновлённые теги
func DelUserTags(id string, tagsForDel []string) ([]string, error) {
	updatedTags := make([]string, 0)

	err := dbAdapter.Update(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))

		userBucket := usersBucket.Bucket([]byte(id))
		if userBucket == nil {
			return errors.New("User with id '" + id + "' doesn't exist")
		}

		// Получение slice из старых тегов
		oldTags, err := toSlice(userBucket.Get([]byte("Tags")))
		if err != nil {
			return err
		}

		// Удаление тегов
		updatedTags = delTags(oldTags, tagsForDel)

		updatedTagsString := strings.Join(updatedTags, " ")
		userBucket.Put([]byte("Tags"), []byte(updatedTagsString))

		return nil
	})
	if err != nil {
		return []string{}, err
	}

	return updatedTags, nil
}

// DelAllUserTags удаляет ВСЕ теги
func DelAllUserTags(id string) error {
	err := dbAdapter.Update(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))

		userBucket := usersBucket.Bucket([]byte(id))
		if userBucket == nil {
			return errors.New("User with id '" + id + "' doesn't exist")
		}

		userBucket.Put([]byte("Tags"), []byte(""))

		return nil
	})

	return err
}

// StopMailout останавливает рассылку для пользователя
func StopMailout(id string) error {
	err := dbAdapter.Update(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))

		userBucket := usersBucket.Bucket([]byte(id))
		if userBucket == nil {
			return errors.New("User with id '" + id + "' doesn't exist")
		}

		userBucket.Put([]byte("Mailout"), []byte("false"))

		return nil
	})

	return err
}
