package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/order", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "{\"status\":\"ok\"}") })
	log.Println("Limit orders on :8102")
	http.ListenAndServe(":8102", mux)
}
