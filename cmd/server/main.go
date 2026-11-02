package main

import (
	"log"
	"net/http"

	"github.com/kaszta1274/macrorecipe/internal/api"
)

const (
	port = "8080"
)

func main() {
	r := api.NewRouter()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}
	log.Fatal(srv.ListenAndServe())
}
