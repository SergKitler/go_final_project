package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func createDb(dbName string) {
	appPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	dbFile := filepath.Join(appPath, dbName)
	_, err = os.Stat(dbFile)

	var install bool
	if err != nil {
		install = true
	}
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()
	if install {
		_, err = db.ExecContext(
			context.Background(),
			`CREATE TABLE IF NOT EXISTS scheduler (
					id INTEGER PRIMARY KEY AUTOINCREMENT, 
					date VARCHAR(8) NOT NULL, 
					title TEXT NOT NULL, 
					comment TEXT NULL, 
					repeat VARCHAR(128) NOT NULL
					);
			 CREATE INDEX id ON scheduler (id)`,
		)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func NextDate(now time.Time, date string, repeat string) (string, error) {
	var (
		err         error
		result_time time.Time
	)
	re := regexp.MustCompile(`^d ([1-9]\d?|1[0-4]\d|400)$`)
	if repeat == "" {
		return "", errors.New("repeat is missing")
	}
	format_time, err := time.Parse("20060102", date)
	if err != nil {
		return "", errors.New("error of parsing date")
	}
	if re.MatchString(repeat) {
		days_str := strings.Fields(repeat)[1]
		days, _ := strconv.Atoi(days_str)

		for {
			result_time = format_time.AddDate(0, 0, days)
			if result_time.After(now) {
				break
			}
		}
	} else if repeat == "y" {
		for {
			result_time = format_time.AddDate(1, 0, 0)
			if result_time.After(now) {
				break
			}
		}
	} else {
		return "", errors.New("repeat has a wrong format")
	}
	return result_time.Format("20060102"), nil

}

func main() {
	createDb("scheduler.db")

	http.Handle("/", http.FileServer(http.Dir("./web")))
	http.HandleFunc("/api/nextdate", func(w http.ResponseWriter, r *http.Request) {
		now_str := r.URL.Query().Get("now")
		now, err := time.Parse("20060102", now_str)
		if err != nil {
			log.Fatal("Wrong formate date")
		}
		date := r.URL.Query().Get("date")
		repeat := r.URL.Query().Get("repeat")
		test_date, err := NextDate(now, date, repeat)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(test_date)
		}

	})

	err := http.ListenAndServe(":7540", nil)
	if err != nil {
		log.Fatal(err)
	}

}
