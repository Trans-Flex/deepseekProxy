package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/sync/singleflight"
)

func main() {
	godotenv.Load()
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	cache, err := NewCache(64 * 1024 * 1024)
	if err != nil {
		log.Fatal(err)
	}
	h := &handler{cache: cache, key: apiKey, sf: &singleflight.Group{}, client: &http.Client{Timeout: 30 * time.Second}}
	var mux = http.NewServeMux()
	mux.HandleFunc("/chat", h.handleChat)
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", mux))
}
