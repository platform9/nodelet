package ipv6

import (
	"fmt"
	"net"
	"reflect"
)

// ValidateString validates that the given string is an IPv4 address
func ValidateString(input string) (bool, error) {
	ip := net.ParseIP(input)

	if ip == nil {
		return false, fmt.Errorf("%s is not an IPv6 address", input)
	}

	if ip.To4() != nil {
		return false, fmt.Errorf("%s is not an IPv6 address", input)
	}

	return true, nil
}

// ValidateStructField validates a struct field holds an IPv4 address
func ValidateStructField(value reflect.Value, tag string) (bool, error) {
	if value.Kind() != reflect.String {
		return false, fmt.Errorf("only strings can be IPv6 validated")
	}

	return ValidateString(value.String())
}
