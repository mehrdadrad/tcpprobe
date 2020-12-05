package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	// HTTPS
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, TCPProbe")
	}))

	r := request{
		count:    2,
		quiet:    true,
		timeout:  time.Second * 2,
		insecure: true,
	}

	c := newClient(&r, ts.URL)
	assert.Equal(t, 2, c.req.count)

	err := c.connect()
	assert.NoError(t, err)
	err = c.httpGet()
	assert.NoError(t, err)
	err = c.getTCPInfo()
	assert.NoError(t, err)
	c.close()

	c = newClient(&r, ts.URL)
	c.probe()

	assert.Equal(t, uint8(1), c.stats.State)
	assert.Equal(t, 200, c.HTTPStatusCode)
	assert.Equal(t, int64(16), c.stats.HTTPRcvdBytes)
	assert.Equal(t, int64(0), c.stats.TCPConnectError)
	assert.Equal(t, int64(0), c.stats.DNSResolveError)
	assert.Equal(t, uint32(0), c.stats.Unacked)
	assert.Equal(t, uint32(0), c.stats.Lost)
	assert.Less(t, uint32(0), c.stats.Rto)
	assert.Less(t, uint32(0), c.stats.Ato)
	assert.Less(t, int64(0), c.stats.TLSHandshake)

	c.close()

	// HTTP
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, TCPProbe")
	}))

	c = newClient(&r, ts.URL)
	err = c.connect()
	assert.NoError(t, err)
	err = c.httpGet()
	assert.NoError(t, err)
	err = c.getTCPInfo()
	assert.NoError(t, err)

	assert.Equal(t, int64(0), c.stats.TLSHandshake)

	c.close()

	// unreachable host
	c = newClient(&r, "127.0.0.0")
	err = c.connect()
	assert.Error(t, err)

	// name not known
	c = newClient(&r, "tcpprobeunknowndomain.com")
	err = c.connect()
	assert.Error(t, err)

	// unreachable ipv6 addr
	c = newClient(&r, "[::1]:5050")
	err = c.connect()
	assert.Error(t, err)

	// wrong target
	c = newClient(&r, ":::")
	err = c.connect()
	assert.Error(t, err)

	// external, without explicit port
	c = newClient(&r, "https://www.google.com")
	err = c.connect()
	assert.NoError(t, err)
	c.close()
}

func TestCli(t *testing.T) {
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	args := []string{"tcpprobe", "-metrics"}
	_, m, err := getCli(args)
	assert.NoError(t, err)
	assert.Len(t, m, 0)
	buf := new(bytes.Buffer)
	io.CopyN(buf, r, 7)
	assert.Equal(t, "metrics", buf.String())

	r, w, _ = os.Pipe()
	os.Stdout = w
	args = []string{"tcpprobe"}
	_, m, err = getCli(args)
	assert.Error(t, err)
	assert.Len(t, m, 0)
	buf.Reset()

	io.CopyN(buf, r, 5)
	assert.Equal(t, "usage", buf.String())

	args = []string{"tcpprobe", "127.0.0.1"}
	_, m, err = getCli(args)
	assert.NoError(t, err)
	assert.Len(t, m, 1)

	os.Stdout = stdout
}

func TestPrometheus(t *testing.T) {
	c := &client{}
	c.prometheus()

	v := reflect.ValueOf(&c.stats).Elem()
	for i := 0; i < v.NumField(); i++ {
		i := i

		req := prometheus.NewCounter(prometheus.CounterOpts{
			Name:        "tp_" + v.Type().Field(i).Tag.Get("name"),
			Help:        v.Type().Field(i).Tag.Get("help"),
			ConstLabels: prometheus.Labels{"target": c.target},
		})

		if err := prometheus.Register(req); err != nil {
			_, ok := err.(prometheus.AlreadyRegisteredError)
			assert.True(t, ok)
		}
	}
}

func TestServerName(t *testing.T) {
	r := request{
		serverName: "myserver",
	}

	c := newClient(&r, "target")
	assert.Equal(t, "myserver", c.serverName())

	c = newClient(&request{}, "target")
	assert.Equal(t, "target", c.serverName())
}

func TestGetSrcAddr(t *testing.T) {
	addr := getSrcAddr("")
	assert.Nil(t, addr)

	addr = getSrcAddr("192.168.1.1")
	assert.Equal(t, &net.TCPAddr{
		IP:   net.ParseIP("192.168.1.1"),
		Port: 0, Zone: "",
	}, addr)
}

func TestPrintText(t *testing.T) {
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	c := &client{stats: stats{Rtt: 5}, req: &request{filter: "rtt"}}
	c.printer(0)

	buf := new(bytes.Buffer)
	io.CopyN(buf, r, 36)
	assert.Contains(t, buf.String(), "Rtt:5")

	os.Stdout = stdout
}

func TestPrintJsonPretty(t *testing.T) {
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	c := &client{stats: stats{}, req: &request{jsonPretty: true}}
	c.printer(0)

	buf := make([]byte, 730)
	n, _ := io.ReadFull(r, buf)
	assert.Equal(t, 730, n)

	os.Stdout = stdout
}

func TestPrintJson(t *testing.T) {
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	c := &client{stats: stats{}, req: &request{json: true}}
	c.printer(0)

	buf := make([]byte, 330)
	n, _ := io.ReadFull(r, buf)
	assert.Equal(t, 330, n)

	os.Stdout = stdout
}

func TestBoolToInt(t *testing.T) {
	assert.Equal(t, 1, boolToInt(true))
	assert.Equal(t, 0, boolToInt(false))
}

func TestNoRedirect(t *testing.T) {
	c := &client{}
	assert.Error(t, c.noRedirect(nil, nil))
}

func TestMain(t *testing.T) {
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, TCPProbe")
	}))
	os.Args = []string{"tcpprobe", "-c", "1", "-insecure", ts.URL}
	main()

	buf := new(bytes.Buffer)
	io.CopyN(buf, r, 500)

	assert.Contains(t, buf.String(), "Target:https://127.0.0.1")
	assert.Contains(t, buf.String(), "HTTPStatusCode:200")

	os.Stdout = stdout
}

func TestIsIPAddr(t *testing.T) {
	assert.True(t, isIPAddr("8.8.8.8"))
	assert.False(t, isIPAddr("www.yahoo.com"))
}
