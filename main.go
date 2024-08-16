package main

import (
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("./web")))
	http.ListenAndServe(":7540", nil)
}
