package utils

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
)

func PostToSlackBestEffort(url string, msg string) {
	body := fmt.Sprintf(`{"text":"%s"}`, msg)
	buf := bytes.NewReader([]byte(body))
	log.Println("message to be posted to Slack: ", msg)
	resp, err := http.Post(url, "application/json", buf)
	if err != nil {
		log.Println("failed to post to slack: ", err)
		return
	}
	status := resp.StatusCode
	if status != 200 {
		log.Println("unexpected slack status code: ", status)
		return
	}
}
