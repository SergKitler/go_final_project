package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func ApiNextDate(w http.ResponseWriter, r *http.Request) {
	now_str := r.URL.Query().Get("now")
	now, err := time.Parse("20060102", now_str)
	if err != nil {
		fmt.Fprintln(w, err)
	} else {
		date := r.URL.Query().Get("date")
		repeat := r.URL.Query().Get("repeat")
		test_date, err := NextDate(now, date, repeat)
		if err != nil {
			fmt.Fprintln(w, err)
		} else {
			fmt.Fprintln(w, test_date)
		}
	}
}

type sql_db struct {
	db *sql.DB
}

type task_json struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment,omitempty"`
	Repeat  string `json:"repeat"`
}

type id_task_response struct {
	Id int64 `json:"id"`
}

func ApiTask(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.ServeFile(w, r, "form.html")
	case "POST":
		decoder := json.NewDecoder(r.Body)
		var task task_json
		err := decoder.Decode(&task)
		if err != nil {
			SendErrorResponse(w, "ApiTask: JSON deserialization error", http.StatusBadRequest)
			return
		}

		if task.Date == "" {
			task.Date = time.Now().Format("20060102")
		}

		var date time.Time
		date, err = time.Parse("20060102", task.Date)
		if err != nil {
			task.Date = time.Now().Format("20060102")
		}

		if date.Before(time.Now()) {
			if task.Repeat == "" {
				task.Date = time.Now().Format("20060102")
			} else {
				task.Date, err = NextDate(time.Now(), task.Date, task.Repeat)

				if err != nil {
					SendErrorResponse(w, "Invalid task repetition format 1", http.StatusBadRequest)
					return
				}
			}
		}

		if task.Title == "" {
			SendErrorResponse(w, "Task title must be specified", http.StatusBadRequest)
			return
		}
		if task.Repeat == "" {
			task.Date, err = NextDate(time.Now(), task.Date, task.Repeat)

			if err != nil {
				SendErrorResponse(w, "Invalid task repetition format", http.StatusBadRequest)
				return
			}
			// else {
			// 	SendErrorResponse(w, "Invalid repeat format", http.StatusBadRequest)
			// 	return
			// }
		}

		db := openDb()
		id, err := db.Add(task)
		if err != nil {
			SendErrorResponse(w, "Failed to write to database", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(id_task_response{Id: id})
	}
}

type error_response struct {
	Error string `json:"error"`
}

func SendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(error_response{Error: message})
}

func openDb() sql_db {
	dbFile := findPathDb(databaseName)

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatal(err)
	}

	return sql_db{db: db}
}

func (at sql_db) Add(task task_json) (int64, error) {
	res, err := at.db.Exec("INSERT INTO scheduler (date, title, comment, repeat) values (?, ?, ?, ?)",
		task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil

}
