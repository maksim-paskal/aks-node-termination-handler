package main

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

const port = ":28080"

// simple server for test env.
func main() {
	http.Handle("/", http.FileServer(http.Dir(".")))
	log.Infof("Listen %s", port)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
