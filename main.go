package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var (
	databaseName string = "scheduler.db"
)

func findPathDb(dbName string) string {
	appPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	dbFile := filepath.Join(appPath, dbName)
	_, err = os.Stat(dbFile)

	return dbFile
}

func СreateDb(dbName string) {
	dbFile := findPathDb(dbName)

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()

	var install bool
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
}

func main() {
	СreateDb(databaseName)

	http.Handle("/", http.FileServer(http.Dir("./web")))
	http.HandleFunc("/api/nextdate", ApiNextDate)
	http.HandleFunc("/api/task", ApiTask)

	err := http.ListenAndServe(":7540", nil)
	if err != nil {
		log.Fatal(err)
	}

}
