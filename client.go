package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

type client struct {
	conn net.Conn
	req  *request

	timing
	httpResponse
}
type httpResponse struct {
	statusCode int
}

type timing struct {
	connect int64
	resolve int64
}

func newClient(req *request) (*client, error) {
	c := &client{req: req}

	return c, c.connect()
}

func (c *client) connect() error {
	var err error

	addr, err := c.getAddr()
	if err != nil {
		return err
	}

	t := time.Now()
	c.conn, err = net.DialTimeout("tcp", addr, c.req.timeout)
	if err != nil {
		return err
	}
	c.timing.connect = time.Since(t).Microseconds()

	return nil
}

func (c *client) dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return c.conn, nil
}

func (c *client) getAddr() (string, error) {
	var host string

	if c.req.urlSchema.Host != "" {
		host = c.req.urlSchema.Host
	} else {
		host = c.req.target
	}

	host, port, err := net.SplitHostPort(host)
	if err != nil {
		return "", err
	}

	t := time.Now()
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}
	c.timing.resolve = time.Since(t).Microseconds()

	for _, addr := range addrs {
		// IPv4 requested
		if !c.req.ipv6 {
			if net.ParseIP(addr).To4() != nil {
				return net.JoinHostPort(addr, port), nil
			}
			continue
		}

		// IPv6 requested
		if net.ParseIP(addr).To4() == nil {
			return net.JoinHostPort(addr, port), nil
		}
	}

	return "", fmt.Errorf("ip address not available")
}

func (c *client) close() {
	c.conn.Close()
}

func (c *client) httpGet() error {
	tr := &http.Transport{
		DialContext:    c.dialContext,
		DialTLSContext: c.dialContext,
	}

	httpClient := &http.Client{
		Transport:     tr,
		CheckRedirect: c.noRedirect,
	}
	resp, err := httpClient.Get(c.req.target)
	if err != nil {
		return err
	}

	io.Copy(ioutil.Discard, resp.Body)

	c.httpResponse.statusCode = resp.StatusCode

	resp.Body.Close()

	return nil
}

func (c *client) noRedirect(req *http.Request, via []*http.Request) error {
	return fmt.Errorf("%s has been redirected", c.req.target)
}
