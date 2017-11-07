package cutlass

import (
	"fmt"
	"net"
	"net/http/httptest"
	"strings"

	"github.com/elazarl/goproxy"
)

func NewTLSProxy() (*httptest.Server, error) {
	return newProxy(true)
}

func NewProxy() (*httptest.Server, error) {
	return newProxy(false)
}

func newProxy(tls bool) (*httptest.Server, error) {
	addr, err := publicIP()
	if err != nil {
		return nil, err
	}
	if strings.Contains(addr, ".") {
		addr = addr + ":0"
	} else if strings.Contains(addr, ":") {
		addr = "[" + addr + "]:0"
	} else {
		return nil, fmt.Errorf("Could not convert address (%s) to address + port", addr)
	}

	ts := httptest.NewUnstartedServer(goproxy.NewProxyHttpServer())
	ts.Listener.Close()
	ts.Listener, err = net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	if tls {
		ts.StartTLS()
	} else {
		ts.Start()
	}
	return ts, nil
}

func publicIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	var addr string
	for _, i := range interfaces {
		if strings.Contains(i.Flags.String(), "up") {
			addrs, err := i.Addrs()
			if err == nil && len(addrs) > 0 {
				addr = addrs[0].String()
			}
		}
	}
	idx := strings.Index(addr, "/")
	if idx > -1 {
		addr = addr[:idx]
	}

	if addr == "" {
		return "", fmt.Errorf("Could not determine IP address")
	}

	return addr, nil
}
