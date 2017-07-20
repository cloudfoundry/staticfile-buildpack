package cutlass

import (
	"fmt"
	"net"
	"net/http/httptest"
	"strings"

	"github.com/elazarl/goproxy"
)

func NewProxy() (*httptest.Server, error) {
	addr, err := publicIP()
	if err != nil {
		return nil, err
	}
	ts := httptest.NewUnstartedServer(goproxy.NewProxyHttpServer())
	ts.Listener.Close()
	ts.Listener, err = net.Listen("tcp", addr+":0")
	if err != nil {
		return nil, err
	}

	ts.Start()
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
