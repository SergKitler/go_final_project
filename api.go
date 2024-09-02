package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
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
	Id      string `json:"id,omitempty`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment,omitempty"`
	Repeat  string `json:"repeat"`
}

type id_task_response struct {
	Id int64 `json:"id"`
}

type tasks_response struct {
	Tasks []task_json `json:"tasks"`
}

func GetTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	const taskLimit = 50
	var (
		taskList []task_json
		task     task_json
		rows     *sql.Rows
		err      error
	)
	db := openDb()
	search_str := r.FormValue("search")
	if search_str != "" {
		var search_date time.Time
		search_date, err = time.Parse("02.01.2006", search_str)
		if err == nil {
			// get tasks by date
			date_res := search_date.Format("20060102")
			query := "SELECT id, date, title, comment, repeat FROM scheduler WHERE date = $1 ORDER BY date LIMIT $2"
			rows, err = db.Get(taskLimit, query, date_res)
			if err != nil {
				SendErrorResponse(w, "Error executing db query", http.StatusInternalServerError)
				return
			}
			defer rows.Close()
		} else {
			// get tasks by title/comment
			search_contain := "%" + search_str + "%"
			query := "SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE $1 OR comment LIKE $1 ORDER BY date LIMIT $2"
			rows, err = db.Get(taskLimit, query, search_contain)
			if err != nil {
				SendErrorResponse(w, "Error executing db query", http.StatusInternalServerError)
				return
			}
			defer rows.Close()
		}
	} else {
		query := "SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT $1"
		rows, err = db.Get(taskLimit, query)
		if err != nil {
			SendErrorResponse(w, "Error executing db query", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
	}

	if err := rows.Err(); err != nil {
		SendErrorResponse(w, "Failed to iterate over rows", http.StatusInternalServerError)
		return
	}

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id, &task.Date, &task.Title, &task.Comment, &task.Repeat); err != nil {
			SendErrorResponse(w, "Error scanning data from the database", http.StatusInternalServerError)
			return
		}
		task.Id = fmt.Sprint(id)
		taskList = append(taskList, task)
	}

	if len(taskList) == 0 {
		taskList = []task_json{}
	}

	// sort tasks
	sort.Slice(taskList, func(i, j int) bool {
		return taskList[i].Date < taskList[j].Date
	})
	responseMap := map[string][]task_json{"tasks": taskList}
	response, err := json.Marshal(responseMap)
	if err != nil {
		SendErrorResponse(w, "Response JSON creation error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func AddTaskPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var task task_json
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		SendErrorResponse(w, "ApiTask: JSON deserialization error", http.StatusBadRequest)
		return
	}
	log.Println("First - date: ", task.Date, "task: ", task.Title, "repeat: ", task.Repeat)

	if task.Title == "" {
		SendErrorResponse(w, "Task title must be specified", http.StatusBadRequest)
		return
	}

	if task.Date == "" {
		task.Date = time.Now().Format("20060102")
	}

	date, err := time.Parse("20060102", task.Date)
	if err != nil {
		SendErrorResponse(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	if task.Repeat != "" {
		date_check, err := NextDate(time.Now(), task.Date, task.Repeat)
		if date_check == "" && err != nil {
			SendErrorResponse(w, "Invalid task repetition format", http.StatusBadRequest)
			return
		}
	}

	if date.Before(time.Now()) {
		if task.Repeat == "" || date.Truncate(24*time.Hour) == date.Truncate(24*time.Hour) {
			task.Date = time.Now().Format("20060102")
		} else {
			date_next, err := NextDate(time.Now(), date.Format("20060102"), task.Repeat)

			if err != nil {
				log.Println("Second - date: ", task.Date, "task: ", task.Title, "repeat: ", task.Repeat)
				SendErrorResponse(w, "Invalid task repetition format 1", http.StatusBadRequest)
				return
			}
			task.Date = date_next
		}
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

func (at sql_db) Get(taskLimit int, args ...string) (*sql.Rows, error) {
	var (
		query  string
		search string
		row    *sql.Rows
		err    error
	)
	if len(args) == 2 {
		query = args[0]
		search = args[1]
		row, err = at.db.Query(query, search, taskLimit)
	} else if len(args) == 1 {
		query = args[0]
		row, err = at.db.Query(query, taskLimit)
	} else {
		return nil, errors.New("Mismatch arguments")
	}

	return row, err
}
