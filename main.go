package main

import (
	"log"
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("./web")))

	err := http.ListenAndServe(":7540", nil)
	if err != nil {
		log.Fatal(err)
	}
}
