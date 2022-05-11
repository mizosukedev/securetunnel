package proxy

import (
	"fmt"
	"strconv"
	"strings"
)

// ServiceConfig is a structure that holds network connection information for each serviceID.
type ServiceConfig struct {
	// ServiceID serviceID
	ServiceID string
	// Network tcp or unix
	Network string
	// Address network address
	Address string
}

// ParseURL returns network and address from given argument string.
// url should be in the following format.
// 	- network://hostname:port ex. tcp://localhost:10022
// 	- hostname:port ex. 127.0.0.1:10022 -> tcp://127.0.0.1:10022 :default network is tcp
// 	- port ex. 10022 -> tcp://localhost:10022 :default hostname is localhost
// In case of unix domain socket, url should be the following format.
// 	- unix://filepath ex. unix:///tmp/securetunnel.sock
func ParseURL(url string) (network string, address string, err error) {

	parts := strings.Split(url, "://")

	switch len(parts) {
	case 1:
		network = "tcp"
		address = parts[0]
	case 2:
		network = parts[0]
		address = parts[1]
	default:
		err = fmt.Errorf("invalid address format. format=%s", address)
		return "", "", err
	}

	network = strings.TrimSpace(network)

	if network == "unix" {
		return network, address, nil
	}

	host := ""
	port := ""

	parts = strings.Split(address, ":")

	switch len(parts) {
	case 1:
		host = "localhost"
		port = parts[0]
	case 2:
		host = parts[0]
		port = parts[1]
	default:
		err := fmt.Errorf("invalid address format. format=%s", address)
		return "", "", err
	}

	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)

	_, err = strconv.ParseUint(port, 10, 16)
	if err != nil {
		err = fmt.Errorf("invalid port number. port=%s", port)
		return "", "", err
	}

	address = fmt.Sprintf("%s:%s", host, port)

	return network, address, nil
}

// ParseServiceConfigs returns []ServiceConfig from given argument string.
// serviceStrs argument is a comma-separated string of service string.
// service string must be in the format "serviceID=address" or "address".
// "serviceID" represents serviceID in AWS secure tunneling service.
func ParseServiceConfig(serviceStr string) ([]ServiceConfig, error) {

	services := strings.Split(serviceStr, ",")
	configs := make([]ServiceConfig, len(services))

	for i, service := range services {

		serviceID := ""
		url := ""

		parts := strings.Split(service, "=")
		switch len(parts) {
		case 1:
			serviceID = ""
			url = parts[0]
		case 2:
			serviceID = parts[0]
			url = parts[1]
		default:
			err := fmt.Errorf("invalid service format -> %s", serviceStr)
			return []ServiceConfig{}, err
		}

		serviceID = strings.TrimSpace(serviceID)
		url = strings.TrimSpace(url)

		network, address, err := ParseURL(url)
		if err != nil {
			return []ServiceConfig{}, err
		}

		config := ServiceConfig{
			ServiceID: serviceID,
			Network:   network,
			Address:   address,
		}

		configs[i] = config
	}

	return configs, nil
}
