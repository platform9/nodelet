package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/apparentlymart/go-cidr/cidr"
)

func main() {
	hostCIDR := flag.String("cidr", "", "IPv4/IPv6 CIDR")
	pos := flag.Int("pos", 0, "<pos>th IP to generate")
	flag.Parse()

	_, ipnet, errCidr := net.ParseCIDR(*hostCIDR)
	if errCidr != nil {
		fmt.Println("None")
		os.Exit(0)
	}
	ip, errPos := cidr.Host(ipnet, *pos)
	if errPos != nil {
		fmt.Println("None")
		os.Exit(0)
	}

	fmt.Println(ip)
}
