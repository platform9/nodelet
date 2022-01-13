package main

import (
	"log"
	"net/http"

	v1 "github.com/platform9/pf9-addons/api/handlers/v1"
	"github.com/platform9/pf9-addons/pkg/k8s"
)

func main() {
	// passing true as argument coz this utility
	// will be running inside k8s pod and it expects
	// serviceaccount token and API address available
	// inside the container.
	client, err := k8s.Newk8sClient(true)
	if err != nil {
		log.Fatalf(err.Error())
	}

	http.HandleFunc("/v1/storage", func(w http.ResponseWriter, req *http.Request) {
		v1.Storage(w, req, client)
	})

	http.HandleFunc("/v1/events", func(w http.ResponseWriter, req *http.Request) {
		v1.Events(w, req, client)
	})

	http.HandleFunc("/v1/healthz", func(w http.ResponseWriter, req *http.Request) {
		v1.Healthz(w, req, client)
	})

	log.Println("Listening on 0.0.0.0:8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
