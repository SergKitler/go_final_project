package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"
)

const taskLimit = 50

var dateFormat string = "20060102"

func ApiNextDate(w http.ResponseWriter, r *http.Request) {
	nowStr := r.URL.Query().Get("now")
	now, err := time.Parse(dateFormat, nowStr)
	if err != nil {
		log.Println(w, err)
	} else {
		date := r.URL.Query().Get("date")
		repeat := r.URL.Query().Get("repeat")
		testDate, err := NextDate(now, date, repeat)
		if err != nil {
			fmt.Fprintln(w, err)
		} else {
			fmt.Fprintln(w, testDate)
		}
	}
}

type taskStruct struct {
	Id      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment,omitempty"`
	Repeat  string `json:"repeat"`
}

func GetTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var (
		taskList []taskStruct
		err      error
	)

	searchStr := r.FormValue("search")
	if searchStr != "" {
		var searchDate time.Time
		searchDate, err = time.Parse("02.01.2006", searchStr)
		if err == nil {
			// get tasks by date
			dateRes := searchDate.Format(dateFormat)
			query := "SELECT id, date, title, comment, repeat FROM scheduler WHERE date == $1 ORDER BY date LIMIT $2"
			taskList, err = Get(taskLimit, query, dateRes)
			if err != nil {
				SendErrorResponse(w, "Error executing db query", http.StatusInternalServerError)
				return
			}
		} else {
			searchContain := "%" + searchStr + "%"
			query := "SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE $1 OR comment LIKE $1 ORDER BY date LIMIT $2"
			taskList, err = Get(taskLimit, query, searchContain)
			if err != nil {
				SendErrorResponse(w, "Error executing db query", http.StatusInternalServerError)
				return
			}
		}
	} else {
		query := "SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT $1"
		taskList, err = Get(taskLimit, query)
		if err != nil {
			SendErrorResponse(w, "Error executing db query", http.StatusInternalServerError)
			return
		}
	}

	if len(taskList) == 0 {
		taskList = []taskStruct{}
	}

	// sort tasks
	sort.Slice(taskList, func(i, j int) bool {
		return taskList[i].Date < taskList[j].Date
	})
	responseMap := map[string][]taskStruct{"tasks": taskList}
	response, err := json.Marshal(responseMap)
	if err != nil {
		SendErrorResponse(w, "Response JSON creation error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response)
}

func Task(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		GetTaskByID(w, r)
	case "POST":
		AddTask(w, r)
	case "PUT":
		EditTask(w, r)
	case "DELETE":
		DelTask(w, r)
	}
}

func AddTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var task taskStruct
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		SendErrorResponse(w, "JSON deserialization error", http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		SendErrorResponse(w, "Task title must be specified", http.StatusBadRequest)
		return
	}

	if task.Date == "" {
		task.Date = time.Now().Format(dateFormat)
	}

	date, err := time.Parse(dateFormat, task.Date)
	if err != nil {
		SendErrorResponse(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	if task.Repeat != "" {
		dateCheck, err := NextDate(time.Now(), task.Date, task.Repeat)
		if dateCheck == "" && err != nil {
			SendErrorResponse(w, "Invalid task repetition format", http.StatusBadRequest)
			return
		}
	}

	if date.Before(time.Now()) {
		if task.Repeat == "" || date.Truncate(24*time.Hour) == date.Truncate(24*time.Hour) {
			task.Date = time.Now().Format(dateFormat)
		} else {
			dateNext, err := NextDate(time.Now(), date.Format(dateFormat), task.Repeat)

			if err != nil {
				SendErrorResponse(w, "Invalid task repetition format 1", http.StatusBadRequest)
				return
			}
			task.Date = dateNext
		}
	}

	id, err := Add(task)
	if err != nil {
		SendErrorResponse(w, "Failed to write to database", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(taskStruct{Id: id})
}

func EditTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var task taskStruct
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		SendErrorResponse(w, "JSON deserialization error", http.StatusBadRequest)
		return
	}

	if task.Id == "" {
		SendErrorResponse(w, "Task ID not found", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(task.Id)
	if err != nil || id <= 0 {
		SendErrorResponse(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		SendErrorResponse(w, "Task title must be specified", http.StatusBadRequest)
		return
	}

	if task.Date == "" {
		task.Date = time.Now().Format(dateFormat)
	}

	date, err := time.Parse(dateFormat, task.Date)
	if err != nil {
		SendErrorResponse(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	if date.Before(time.Now()) {
		if task.Repeat == "" || date.Truncate(24*time.Hour) == date.Truncate(24*time.Hour) {
			task.Date = time.Now().Format(dateFormat)
		} else {
			dateNext, err := NextDate(time.Now(), date.Format(dateFormat), task.Repeat)

			if err != nil {
				SendErrorResponse(w, "Invalid task repetition format 1", http.StatusBadRequest)
				return
			}
			task.Date = dateNext
		}
	}

	if task.Repeat != "" {
		if _, err := strconv.Atoi(task.Repeat[2:]); err != nil || (task.Repeat[0] != 'd' && task.Repeat[0] != 'y') {
			SendErrorResponse(w, "Invalid task repetition format", http.StatusBadRequest)
			return
		}
	}

	var idTask int

	err = SearchError(task.Id, idTask)
	if err == sql.ErrNoRows {
		SendErrorResponse(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		SendErrorResponse(w, "Error checking task existence", http.StatusInternalServerError)
		return
	}

	_, err = Update(task)
	if err != nil {
		SendErrorResponse(w, "Task not found", http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(struct{}{})
	if err != nil {
		SendErrorResponse(w, "Response JSON creation error", http.StatusInternalServerError)
		return
	}
	// send response
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response)
}

func GetTaskByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendErrorResponse(w, "Method not allowed", http.StatusBadRequest)
		return
	}

	idTask := r.FormValue("id")
	if idTask == "" {
		SendErrorResponse(w, "No ID provided", http.StatusBadRequest)
		return
	}

	var task taskStruct

	task, err := GetbyID(idTask)
	if err == sql.ErrNoRows {
		SendErrorResponse(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		SendErrorResponse(w, "Error executing db query", http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(task)
	if err != nil {
		SendErrorResponse(w, "Response JSON creation eror", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response)
}

func DelTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		SendErrorResponse(w, "Method not allowed", http.StatusBadRequest)
		return
	}

	idTask := r.FormValue("id")
	if idTask == "" {
		SendErrorResponse(w, "No ID provided", http.StatusBadRequest)
		return
	}

	resultId, err := strconv.Atoi(idTask)
	if err != nil {
		SendErrorResponse(w, "Invalid ID format", http.StatusInternalServerError)
		return
	}

	res, err := Delete(resultId)

	if err != nil {
		SendErrorResponse(w, "Error deleting task", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		SendErrorResponse(w, "Unable to determine the number of affected rows", http.StatusInternalServerError)
		return
	} else if rowsAffected == 0 {
		SendErrorResponse(w, "Task not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{}`))
}

func TaskDone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendErrorResponse(w, "Method not allowed", http.StatusBadRequest)
		return
	}

	idTask := r.FormValue("id")
	if idTask == "" {
		SendErrorResponse(w, "No ID provided", http.StatusBadRequest)
		return
	}

	var (
		task taskStruct
		id   int
	)

	id, task, err := GetbyIdWithId(idTask)
	task.Id = fmt.Sprint(id)
	if err == sql.ErrNoRows {
		SendErrorResponse(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		SendErrorResponse(w, "Error retrieving task data", http.StatusInternalServerError)
		return
	}

	if task.Repeat != "" {
		newTaskDate, err := NextDate(time.Now(), task.Date, task.Repeat)
		if err != nil {
			SendErrorResponse(w, "Invalid repeat pattern", http.StatusBadRequest)
			return
		}

		task.Date = newTaskDate

		_, err = Update(task)
		if err != nil {
			SendErrorResponse(w, "Task not found", http.StatusInternalServerError)
			return
		}
	} else {
		// delete task if repeat rule not set
		res, err := Delete(id)
		if err != nil {
			SendErrorResponse(w, "Error deleting task", http.StatusInternalServerError)
			return
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			SendErrorResponse(w, "Unable to determine the number of rows affected after deleting a task", http.StatusInternalServerError)
			return
		} else if rowsAffected == 0 {
			SendErrorResponse(w, "Task not found", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{}`))
}

type errorResponse struct {
	Error string `json:"error"`
}

func SendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse{Error: message})
}
