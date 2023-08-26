package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/ping1", handler1)
	http.HandleFunc("/ping2", handler2)
	http.HandleFunc("/ping3", handler3)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}

func handler1(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusFailedDependency)
	_, err := w.Write([]byte("request /ping1"))
	if err != nil {
		return
	}
}

func handler2(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusFailedDependency)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, err := w.Write([]byte("request /ping2"))
	if err != nil {
		return
	}
}

func handler3(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, err := w.Write([]byte("request /ping3"))
	if err != nil {
		return
	}
	w.WriteHeader(http.StatusFailedDependency)
}
