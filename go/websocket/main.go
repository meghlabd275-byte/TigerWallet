package main

import (
	"log"
	"net/http"
)

func main() {
	log.Println("WebSocket server on :8098")
	http.ListenAndServe(":8098", nil)
}
