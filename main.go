package main

import (
	"log"
	"net/http"

	_ "modernc.org/sqlite"
)

var databaseName string = "scheduler.db"

type ApiHandler struct {
	db sql_db
}

func main() {
	var handler ApiHandler
	handler.db.СreateDb(databaseName)
	handler.db.openDb()

	m := http.NewServeMux()
	m.Handle("/", http.FileServer(http.Dir("./web")))
	m.HandleFunc("/api/nextdate", ApiNextDate)
	m.HandleFunc("/api/task", handler.Task)
	m.HandleFunc("/api/tasks", handler.GetTasks)
	m.HandleFunc("/api/task/done", handler.TaskDone)

	var srv = &http.Server{
		Addr:    ":7540",
		Handler: m,
	}
	log.Println("test")
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
