package middlewares

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/eser/ajan/httpfx"
)

const (
	ClientAddr       httpfx.ContextKey = "client-addr"
	ClientAddrIp     httpfx.ContextKey = "client-addr-ip"
	ClientAddrOrigin httpfx.ContextKey = "client-addr-origin"
)

var ErrInvalidIPAddress = errors.New("invalid IP address")

func ResolveAddressMiddleware() httpfx.Handler {
	return func(ctx *httpfx.Context) httpfx.Result {
		addr := GetClientAddrs(ctx.Request)

		newContext := context.WithValue(
			ctx.Request.Context(),
			ClientAddr,
			addr,
		)

		isLocal, err := DetectLocalNetwork(addr)
		if err != nil {
			return ctx.Results.Error(http.StatusInternalServerError, []byte(err.Error()))
		}

		if isLocal {
			newContext = context.WithValue(
				newContext,
				ClientAddrOrigin,
				"local",
			)

			ctx.ResponseWriter.Header().
				Set("X-Request-Origin", "local: "+addr)

			ctx.UpdateContext(newContext)

			return ctx.Next()
		}

		// TODO(@eser) add ip allowlist and blocklist implementations

		newContext = context.WithValue(
			newContext,
			ClientAddrOrigin,
			"remote",
		)

		ctx.ResponseWriter.Header().
			Set("X-Request-Origin", addr)

		ctx.UpdateContext(newContext)

		return ctx.Next()
	}
}

func splitHostPort(addr string) (string, string, error) {
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

	requestIp, _, err := splitHostPort(requestAddrs[0])
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

func GetClientAddrs(req *http.Request) string {
	requester, hasHeader := req.Header["True-Client-IP"] //nolint:staticcheck

	if !hasHeader {
		requester, hasHeader = req.Header["X-Forwarded-For"]
	}

	if !hasHeader {
		requester, hasHeader = req.Header["X-Real-IP"] //nolint:staticcheck
	}

	// if the requester is still empty, use the hard-coded address from the socket
	if !hasHeader {
		requester = []string{req.RemoteAddr}
	}

	// split comma delimited list into a slice
	// (this happens when proxied via elastic load balancer then again through nginx)
	var addrs []string

	for _, addr := range requester {
		for entry := range strings.SplitSeq(addr, ",") {
			addrs = append(addrs, strings.Trim(entry, " "))
		}
	}

	return strings.Join(addrs, ", ")
}
