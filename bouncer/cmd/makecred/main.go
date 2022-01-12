package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s username password\n", os.Args[0])
	}
	j := fmt.Sprintf(`{"username": "%s", "password": "%s"}`, os.Args[1], os.Args[2])
	c := base64.StdEncoding.EncodeToString([]byte(j))
	fmt.Println(c)
	d, _ := base64.StdEncoding.DecodeString(c)
	if string(d) != j {
		log.Fatalln("unexpected decoded value: ", string(d))
	}
}
