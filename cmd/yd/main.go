package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/ilya-pirogov/youtube-downloader/pkg"
	"log"
	"net/http"
	"time"
)

var port int

func init() {
	flag.IntVar(&port, "port", 80, "http port")
}

func main() {
	flag.Parse()

	r := mux.NewRouter()
	d := pkg.NewDispatcher()

	r.HandleFunc("/", d.IndexHandler).Methods(http.MethodGet)
	r.HandleFunc("/add", d.AddToDownload).Methods(http.MethodPost)
	r.HandleFunc("/get", d.GetProgress).Methods(http.MethodGet)
	r.HandleFunc("/ws", d.ServeWs)
	r.PathPrefix("/result/").Handler(http.StripPrefix("/result/", http.FileServer(http.Dir("./out"))))


	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("0.0.0.0:%d", port),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go d.Start()
	log.Fatal(srv.ListenAndServe())
}
