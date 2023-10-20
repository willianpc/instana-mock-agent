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

	var sp []span

	err = json.Unmarshal(b, &sp)

	if err != nil {
		log.Printf("error unmarshalling span: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	mu.Lock()
	dumpedSpans = append(dumpedSpans, sp...)
	mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

func discoveryHandler(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)

	if err != nil {
		log.Printf("error reading discovery request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var discoveryReq discoveryRequest

	err = json.Unmarshal(b, &discoveryReq)

	if err != nil {
		log.Printf("error unmarshalling discovery request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res := discoveryResponse{}
	b, err = json.Marshal(res)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(b)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func dumpHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	b, err := json.Marshal(dumpedSpans)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(b)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
