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

type sql_db struct {
	db *sql.DB
}

func findPathDb(dbName string) string {
	appPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	dbFile := filepath.Join(appPath, dbName)

	return dbFile
}

func (at *sql_db) СreateDb(dbName string) {
	var (
		err     error
		install bool
	)

	dbFile := findPathDb(dbName)

	at.db, err = sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer at.db.Close()

	_, err = os.Stat(dbFile)

	if err != nil {
		install = true
	}

	if install {
		_, err = at.db.ExecContext(
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
}

func (at *sql_db) openDb() {
	dbFile := findPathDb(databaseName)
	var err error
	at.db, err = sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatal(err)
	}
}

func (at *sql_db) Add(task taskStruct) (string, error) {
	res, err := at.db.Exec("INSERT INTO scheduler (date, title, comment, repeat) values (?, ?, ?, ?)",
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

func (at *sql_db) Get(taskLimit int, args ...string) (*sql.Rows, error) {
	var (
		query  string
		search string
		row    *sql.Rows
		err    error
	)
	switch len(args) {
	case 1:
		query = args[0]
		row, err = at.db.Query(query, taskLimit)
	case 2:
		query = args[0]
		search = args[1]
		row, err = at.db.Query(query, search, taskLimit)
	default:
		return nil, errors.New("mismatch arguments")
	}
	return row, err
}

func (at *sql_db) SearchError(query string, id string, id_task int) error {
	err := at.db.QueryRow(query, id).Scan(&id_task)
	return err
}

func (at *sql_db) GetbyID(query string, id string) (taskStruct, error) {
	var task taskStruct
	err := at.db.QueryRow(query, id).Scan(&task.Id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	return task, err
}

func (at *sql_db) GetbyIdWithId(id string) (int, taskStruct, error) {
	var (
		task   taskStruct
		id_res int
	)
	query := "SELECT id, date, title, comment, repeat FROM scheduler WHERE id == ?"
	err := at.db.QueryRow(query, id).Scan(&id_res, &task.Date, &task.Title, &task.Comment, &task.Repeat)

	return id_res, task, err
}

func (at *sql_db) Update(task taskStruct) (sql.Result, error) {
	query := "UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat =? WHERE id == ?"
	res, err := at.db.Exec(query, task.Date, task.Title, task.Comment, task.Repeat, task.Id)
	if err != nil {
		return nil, errors.New("task not found")
	}

	return res, nil
}

func (at *sql_db) Delete(id int) (sql.Result, error) {
	query := "DELETE FROM scheduler WHERE id == ?"
	result, err := at.db.Exec(query, id)

	return result, err
}
