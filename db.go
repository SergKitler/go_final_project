package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

var db *sql.DB

func findPathDb(dbName string) string {
	appPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	dbFile := filepath.Join(appPath, dbName)

	return dbFile
}

func СreateDb(dbName string) *sql.DB {
	var (
		err     error
		install bool
	)

	dbFile := findPathDb(dbName)

	db, err = sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = os.Stat(dbFile)

	if err != nil {
		install = true
	}

	if install {
		_, err = db.ExecContext(
			context.Background(),
			`CREATE TABLE IF NOT EXISTS scheduler (
					id INTEGER PRIMARY KEY AUTOINCREMENT, 
					date VARCHAR(8) NOT NULL, 
					title TEXT NOT NULL, 
					comment TEXT NOT NULL DEFAULT "", 
					repeat VARCHAR(128) NOT NULL DEFAULT ""
					);
			 CREATE INDEX id ON scheduler (id)`,
		)
		if err != nil {
			log.Fatal(err)
		}
	}
	return db
}

func OpenDb() {
	dbFile := findPathDb(databaseName)
	var err error
	db, err = sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatal(err)
	}
}

func Add(task taskStruct) (string, error) {
	res, err := db.Exec("INSERT INTO scheduler (date, title, comment, repeat) values (?, ?, ?, ?)",
		task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		return "", err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return "", err
	}

	return strconv.FormatInt(id, 10), nil
}

func Get(taskLimit int, args ...string) ([]taskStruct, error) {
	var (
		query  string
		search string
		res    []taskStruct
		rows   *sql.Rows
		err    error
	)
	switch len(args) {
	case 1:
		query = args[0]
		rows, err = db.Query(query, taskLimit)
	case 2:
		query = args[0]
		search = args[1]
		rows, err = db.Query(query, search, taskLimit)
	default:
		return nil, errors.New("mismatch arguments")
	}

	defer rows.Close()

	for rows.Next() {
		task := taskStruct{}

		err := rows.Scan(&task.Id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			return nil, err
		}

		res = append(res, task)
	}
	return res, err
}

func SearchError(id string, id_task int) error {
	query := "SELECT id FROM scheduler WHERE id == ?"
	err := db.QueryRow(query, id).Scan(&id_task)
	return err
}

func GetbyID(id string) (taskStruct, error) {
	var task taskStruct
	query := "SELECT id, date, title, comment, repeat FROM scheduler WHERE id == ?"
	err := db.QueryRow(query, id).Scan(&task.Id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	return task, err
}

func GetbyIdWithId(id string) (int, taskStruct, error) {
	var (
		task   taskStruct
		id_res int
	)
	query := "SELECT id, date, title, comment, repeat FROM scheduler WHERE id == ?"
	err := db.QueryRow(query, id).Scan(&id_res, &task.Date, &task.Title, &task.Comment, &task.Repeat)

	return id_res, task, err
}

func Update(task taskStruct) (sql.Result, error) {
	query := "UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat =? WHERE id == ?"
	res, err := db.Exec(query, task.Date, task.Title, task.Comment, task.Repeat, task.Id)
	if err != nil {
		return nil, errors.New("task not found")
	}

	return res, nil
}

func Delete(id int) (sql.Result, error) {
	query := "DELETE FROM scheduler WHERE id == ?"
	result, err := db.Exec(query, id)

	return result, err
}
