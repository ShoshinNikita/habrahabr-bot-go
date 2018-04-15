package db

import (
	"errors"
	"github.com/boltdb/bolt"
	"strings"
)

/*
*	Структура базы данных
*
*	"users"
*		|-> id
*			| HabrTags
*			| HabrMailout
*			| GeekTags
*			| GeekMailout
*
 */

const habr = "habrahabr"
const geek = "geektimes"

// User содержит в себе информацию о пользователе
type User struct {
	ID			int64		`json:"id"`
	HabrTags	[]string	`json:"habrTags"`
	HabrMailout bool		`json:"habrMailout"`
	GeekTags	[]string 	`json:"geekTags"`
	GeekMailout bool		`json:"geekMailout"`
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

/* USERS */

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
			userBucket.Put([]byte("HabrTags"), []byte(""))
			userBucket.Put([]byte("HabrMailout"), []byte("true"))
			userBucket.Put([]byte("GeekTags"), []byte(""))
			userBucket.Put([]byte("GeekMailout"), []byte("true"))
		} else {
			// Если пользователь существовал, то просто включаем ему рассылку
			userBucket.Put([]byte("HabrMailout"), []byte("true"))
			userBucket.Put([]byte("GeekMailout"), []byte("true"))
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
		user.HabrTags, err = toSlice(userBucket.Get([]byte("HabrTags")))
		if err != nil {
			return err
		}
		user.HabrMailout, err = toBool(userBucket.Get([]byte("HabrMailout")))
		if err != nil {
			return err
		}
		user.GeekTags, err = toSlice(userBucket.Get([]byte("GeekTags")))
		if err != nil {
			return err
		}
		user.GeekMailout, err = toBool(userBucket.Get([]byte("GeekMailout")))
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
			user.HabrTags, err = toSlice(userBucket.Get([]byte("HabrTags")))
			if err != nil {
				continue
			}
			user.HabrMailout, err = toBool(userBucket.Get([]byte("HabrMailout")))
			if err != nil {
				continue
			}
			user.GeekTags, err = toSlice(userBucket.Get([]byte("GeekTags")))
			if err != nil {
				continue
			}
			user.GeekMailout, err = toBool(userBucket.Get([]byte("GeekMailout")))
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
func GetUsersNumber() (int64) {
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

/* TAGS */

// AddUserTags добавляет теги, которые были переданы
// Возвращает slice, содержащий обновлённые теги
func AddUserTags(id, site string, newTags []string) ([]string, error) {
	var destination string
	if site == habr {
		destination = "HabrTags"
	} else if site == geek {
		destination = "GeekTags"
	} else {
		return []string{}, errors.New("Bad site")
	}

	updatedTags := make([]string, 0)

	err := dbAdapter.Update(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))

		userBucket := usersBucket.Bucket([]byte(id))
		if userBucket == nil {
			return errors.New("User with id '" + id + "' doesn't exist")
		}
		oldTags := make([]string, 0)

		// Получение slice из старых тегов
		oldTags, err := toSlice(userBucket.Get([]byte(destination)))
		if err != nil {
			return err
		}

		// Добавление тегов
		updatedTags = addTags(oldTags, newTags)

		updatedTagsString := strings.Join(updatedTags, " ")
		userBucket.Put([]byte(destination), []byte(updatedTagsString))

		return nil
	})
	if err != nil {
		return []string{}, err
	}

	return updatedTags, nil
}


// UpdateTags перезаписывает теги
func UpdateTags(id, site string, tags []string) error {
	err := dbAdapter.Update(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))

		userBucket := usersBucket.Bucket([]byte(id))
		if userBucket == nil {
			return errors.New("User with id '" + id + "' doesn't exist")
		}

		tagsString := strings.Join(tags, " ")

		if site == habr {
			userBucket.Put([]byte("HabrTags"), []byte(tagsString))
		} else if site == geek {
			userBucket.Put([]byte("GeekTags"), []byte(tagsString))
		} else {
			return errors.New("Bad site")
		}

		return nil
	})

	return err
}


// DelUserTags удаляет теги, которые были переданы
// Возвращает slice, содержащий обновлённые теги
func DelUserTags(id, site string, tagsForDel []string) ([]string, error) {
	var destination string
	if site == habr {
		destination = "HabrTags"
	} else if site == geek {
		destination = "GeekTags"
	} else {
		return []string{}, errors.New("Bad site")
	}

	updatedTags := make([]string, 0)

	err := dbAdapter.Update(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))

		userBucket := usersBucket.Bucket([]byte(id))
		if userBucket == nil {
			return errors.New("User with id '" + id + "' doesn't exist")
		}
		oldTags := make([]string, 0)

		// Получение slice из старых тегов
		oldTags, err := toSlice(userBucket.Get([]byte(destination)))
		if err != nil {
			return err
		}

		// Удаление тегов
		updatedTags = delTags(oldTags, tagsForDel)

		updatedTagsString := strings.Join(updatedTags, " ")
		userBucket.Put([]byte(destination), []byte(updatedTagsString))

		return nil
	})
	if err != nil {
		return []string{}, err
	}

	return updatedTags, nil
}

// DelAllUserTags удаляет ВСЕ теги
func DelAllUserTags(id, site string) error {
	err := dbAdapter.Update(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))

		userBucket := usersBucket.Bucket([]byte(id))
		if userBucket == nil {
			return errors.New("User with id '" + id + "' doesn't exist")
		}

		if site == habr {
			userBucket.Put([]byte("HabrTags"), []byte(""))
		} else if site == geek {
			userBucket.Put([]byte("GeekTags"), []byte(""))
		} else {
			return errors.New("Bad site")
		}

		return nil
	})

	return err
}

/* MAILOUT */

// StartMailout включает рассылку для пользователя
func StartMailout(id, site string) error {
	err := dbAdapter.Update(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))

		userBucket := usersBucket.Bucket([]byte(id))
		if userBucket == nil {
			return errors.New("User with id '" + id + "' doesn't exist")
		}

		if site == habr {
			userBucket.Put([]byte("HabrMailout"), []byte("true"))
		} else if site == geek {
			userBucket.Put([]byte("GeekMailout"), []byte("true"))
		} else {
			return errors.New("Bad site")
		}

		return nil
	})

	return err
}

// StopMailout останавливает рассылку для пользователя
func StopMailout(id, site string) error {
	err := dbAdapter.Update(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))

		userBucket := usersBucket.Bucket([]byte(id))
		if userBucket == nil {
			return errors.New("User with id '" + id + "' doesn't exist")
		}

		if site == habr {
			userBucket.Put([]byte("HabrMailout"), []byte("false"))
		} else if site == geek {
			userBucket.Put([]byte("GeekMailout"), []byte("false"))
		} else {
			return errors.New("Bad site")
		}

		return nil
	})

	return err
}
