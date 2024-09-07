package main

import (
	"log"
	"net/http"

	_ "modernc.org/sqlite"
)

var (
	databaseName string = "scheduler.db"
)

func main() {
	СreateDb(databaseName)
	OpenDb()

	m := http.NewServeMux()
	m.Handle("/", http.FileServer(http.Dir("./web")))
	m.HandleFunc("/api/nextdate", ApiNextDate)
	m.HandleFunc("/api/task", Task)
	m.HandleFunc("/api/tasks", GetTasks)
	m.HandleFunc("/api/task/done", TaskDone)

	var srv = &http.Server{
		Addr:    ":7540",
		Handler: m,
	}
	log.Println("Starting Server on port 7540")
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
