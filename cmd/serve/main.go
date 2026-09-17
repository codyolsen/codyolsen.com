package main

import (
	"flag"
	"log"
	"net/http"
	"time"
)

func main() {
	port := flag.String("port", "8000", "local server port")
	flag.Parse()

	mux := http.NewServeMux()
	mux.Handle("/game/", http.FileServer(http.Dir("static")))
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/game/", http.StatusFound)
	})

	server := &http.Server{
		Addr:              ":" + *port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
