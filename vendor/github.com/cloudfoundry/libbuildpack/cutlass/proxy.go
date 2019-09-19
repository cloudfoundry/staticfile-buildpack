package cutlass

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http/httptest"
	"net/url"

	"github.com/elazarl/goproxy"
)

/*
cfbuildpacks/proxy-server is a compiled version of this code,
with the Dockerfile below.

To build, use:
`GOOS="linux" go build -o /tmp/proxy-linux ~/workspace/libbuildpack/cutlass/proxy.go`

To create the docker image, use:
FROM bash
ADD proxy-linux /
CMD ["/proxy-linux"]

*/

func main() {
	p, err := NewProxy()
	if err != nil {
		fmt.Printf("Errored out: %s\n", err)
	}
	defer p.Close()

	proxyURL, err := url.Parse(p.URL)
	listenMsg := fmt.Sprintf("Listening on Port: %s", proxyURL.Port())
	fmt.Println(listenMsg)
	if err := ioutil.WriteFile("server.log", []byte(listenMsg), 0644); err != nil {
		fmt.Println("Failed to write to log")
	}

	for {
		continue
	}
}

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
