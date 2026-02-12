package main

import (
	"log"
	"net/http"
	"time"
)

func middlewareLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("START %s %s ", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("END   %s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
