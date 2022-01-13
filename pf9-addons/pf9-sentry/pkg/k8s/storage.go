package k8s

import (
	"encoding/json"
	"log"

	v1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DriversResponse ...
type DriversResponse struct {
	Count int `json:"count"`
	Drivers []v1beta1.CSIDriver `json:"drivers"`

}

// GetCSIDrivers returns a list of CSI drivers
func (c *Client)GetCSIDrivers() []byte {
	driversList, err :=  c.client.StorageV1beta1().CSIDrivers().List(metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
	}

	respData := DriversResponse{
		Count: len(driversList.Items),
		Drivers: driversList.Items,
	}
	byteData, err := json.Marshal(respData)
	if err != nil {
		log.Println(err.Error())
	}

	return byteData
}