package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		// ideally we should print an error but the original python binary that this is going to
		// replace did not. Keeping the golang binary in sync with python one for now.
		os.Exit(1)
	}
	// 0th param is the binary name
	toParse := os.Args[1]
	checkIPAddress(toParse)
}

func checkIPAddress(ip string) {
	if net.ParseIP(ip) == nil {
		// nil means that IP could not be parsed so it must be a hostname i.e. DNS name
		fmt.Println("DNS")
	} else {
		fmt.Println("IP")
	}
}
