package lib

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

var ErrInvalidIPAddress = errors.New("invalid IP address")

func SplitHostPort(addr string) (string, string, error) {
	if !strings.ContainsRune(addr, ':') {
		return addr, "", nil
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", "", fmt.Errorf("failed to split host and port: %w", err)
	}

	return host, port, nil
}

func DetectLocalNetwork(requestAddr string) (bool, error) {
	var requestIp string

	requestAddrs := strings.SplitN(requestAddr, ",", 2) //nolint:mnd

	requestIp, _, err := SplitHostPort(requestAddrs[0])
	if err != nil {
		return false, err
	}

	requestIpNet := net.ParseIP(requestIp)
	if requestIpNet == nil {
		return false, fmt.Errorf("%w - %q", ErrInvalidIPAddress, requestIp)
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false, err //nolint:wrapcheck
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		if !ipNet.Contains(requestIpNet) {
			continue
		}

		if requestIpNet.IsLoopback() {
			return true, nil
		}
	}

	return false, nil
}
