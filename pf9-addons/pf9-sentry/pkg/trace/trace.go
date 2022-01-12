package trace

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Trace ...
type Trace struct {
	Client   string      `json:"clientIP"`
	Header   http.Header `json:"headers"`
	Method   string      `json:"method"`
	URI      string      `json:"uri"`
	Duration string      `json:"duration"`
}

// Track ...
func Track(req *http.Request) (*http.Request, time.Time) {
	return req, time.Now()
}

// Duration ...
func Duration(req *http.Request, t time.Time) {
	duration := time.Since(t)
	tr := Trace{
		Client:   req.RemoteAddr,
		Header:   req.Header,
		Method:   req.Method,
		URI:      req.RequestURI,
		Duration: fmt.Sprintf("%v", duration),
	}

	tByte, _ := json.Marshal(tr)
	log.Printf("TRACE: %s\n", string(tByte))
}
