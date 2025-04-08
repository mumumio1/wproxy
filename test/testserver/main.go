package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/test", handler)
	http.HandleFunc("/cache-test", cacheHandler)
	http.HandleFunc("/slow", slowHandler)

	fmt.Println("Test server listening on :19000")
	if err := http.ListenAndServe(":19000", nil); err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"path":      r.URL.Path,
		"method":    r.Method,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func cacheHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Header().Set("Content-Type", "application/json")
	
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"cacheable": true,
	}

	json.NewEncoder(w).Encode(response)
}

func slowHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(2 * time.Second)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "slow response",
	})
}

