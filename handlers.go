package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

func spanHandler(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)

	if err != nil {
		log.Printf("error reading request body: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	defer r.Body.Close()

	var sp span

	err = json.Unmarshal(b, &sp)

	if err != nil {
		log.Printf("error unmarshalling span: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Println(sp)

	w.WriteHeader(http.StatusOK)
}
