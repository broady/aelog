package main

import (
	"log"
	"net/http"
	"os"

	"github.com/broady/aelog"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", home)

	handler, err := aelog.WrapHandler(mux, "app_log")
	if err != nil {
		log.Fatalf("aelog.WrapHandler: %v", err)
	}

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	http.ListenAndServe(":"+port, handler)
}

func home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	aelog.Infof(ctx, "hello! %v", r.URL.Path)
}
