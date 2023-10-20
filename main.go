package main

import (
	"io"
	"net/http"
)

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "welcome to the mock agent\n")
	})

	http.HandleFunc("/span", spanHandler)

	http.ListenAndServe(":9090", nil)
}
