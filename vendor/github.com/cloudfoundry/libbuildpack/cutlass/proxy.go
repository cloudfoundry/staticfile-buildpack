package cutlass

import (
	"net"
	"net/http/httptest"

	"github.com/elazarl/goproxy"
)

func NewTLSProxy() (*httptest.Server, error) {
	return newProxy(true)
}

func NewProxy() (*httptest.Server, error) {
	return newProxy(false)
}

func newProxy(tls bool) (*httptest.Server, error) {
	var err error
	ts := httptest.NewUnstartedServer(goproxy.NewProxyHttpServer())
	ts.Listener.Close()
	ts.Listener, err = net.Listen("tcp", "0.0.0.0:0")
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
