package articlesdb_test

import (
	"time"
	"testing"
	db "articlesdb"
)

var path = "../../data/articles.db"

func TestOpen(t *testing.T) {
	err := db.Open(path)
	if err != nil {
		t.Error(err)
	}
	err = db.Close()
	if err != nil {
		t.Error(err)
	}
}

func TestGet(t *testing.T) {
	err := db.Open(path)
	if err != nil {
		t.Error(err)
	}

	type Test struct {
		text	string
		key		string
		err		error
	} 
	
	tests := []Test{
		{"Hello", "", nil},
		{"Как дела?", "", nil},
		{"How are you?", "", nil},
		{"Новый подход позволит увеличить скорость работы программистов в 500 раз!", "", nil},
	}

	// добавляем разные значения
	for i, test := range tests {
		tests[i].key, tests[i].err = db.Add(test.text, time.Now().Unix())
	}

	// проверяем
	for _, test := range tests {
		if test.err != nil {
			t.Fatalf("Получена ошибка %s", test.err.Error())
		}
		res, err := db.Get(test.key)
		if err != nil {
			t.Fatalf("Получена ошибка %s", err.Error())
		}
		if test.text != res {
			t.Fatalf("Тексты не совпадают. Должно быть: %s Получено: %s", test.text, res)
		}
	}

	// чистим базу данных
	db.ClearAll()
	db.Close()
}

func TestClear(t *testing.T) {
	db.Open(path)
	err := db.ClearAll()
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
}