package v1

import (
	"net/http"

	"github.com/platform9/pf9-addons/pkg/k8s"
	"github.com/platform9/pf9-addons/pkg/trace"
)

// Storage fetches storage details of k8s cluster
func Storage(w http.ResponseWriter, req *http.Request, client *k8s.Client) {
	defer trace.Duration(trace.Track(req))
	if req.Method != "GET" {
		w.WriteHeader(405)
		return
	}

	respData := client.GetCSIDrivers()
	w.Header().Add("Content-Type", "application/json")
	w.Write(respData)
}

// Events fetches events of k8s cluster
func Events(w http.ResponseWriter, req *http.Request, client *k8s.Client) {
	defer trace.Duration(trace.Track(req))
	if req.Method != "GET" {
		w.WriteHeader(405)
		return
	}

	respData := client.GetEvents()
	w.Header().Add("Content-Type", "application/json")
	w.Write(respData)
}

// Healthz exposes health endpoint
func Healthz(w http.ResponseWriter, req *http.Request, client *k8s.Client) {
	defer trace.Duration(trace.Track(req))
	if req.Method != "GET" {
		w.WriteHeader(405)
		return
	}

	if !client.Ping() {
		w.WriteHeader(404)
		return
	}

	w.Write([]byte("OK"))
}
