package k8s

import (
	"encoding/json"
	"log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventsResponse ...
type EventsResponse struct {
	Count  int        `json:"count"`
	Events []v1.Event `json:"events"`
}

// GetEvents gets events from k8s cluster
func (c *Client) GetEvents() []byte {
	// Empty string as argument to fetch events from all namespaces
	events, err := c.client.CoreV1().Events("").List(metav1.ListOptions{
		//	FieldSelector: "involvedObject.kind=Service",
	})
	if err != nil {
		log.Println(err.Error())
	}

	respData := EventsResponse{
		Count:  len(events.Items),
		Events: events.Items,
	}

	byteData, err := json.Marshal(respData)
	if err != nil {
		log.Println(err.Error())
	}

	return byteData
}
