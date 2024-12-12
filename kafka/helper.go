package kafka

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func ConsumerGroup(component string) string {
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		mac, err := getMacAddr()
		if err != nil {
			return component
		}
		hostname = strings.Join(mac, ":")
	}
	return fmt.Sprintf("%s-%s", hostname, component)
}

func getMacAddr() ([]string, error) {
	ifas, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var as []string
	for _, ifa := range ifas {
		a := ifa.HardwareAddr.String()
		if a != "" {
			as = append(as, a)
		}
	}
	return as, nil
}
