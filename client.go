package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"syscall"
	"time"
	"unsafe"
)

type stats struct {
	State        uint8  `key:"State"`
	CaState      uint8  `key:"Ca_state"`
	Retransmits  uint8  `key:"Retransmits"`
	Probes       uint8  `key:"Probes"`
	Backoff      uint8  `key:"Backoff"`
	Options      uint8  `key:"Options"`
	Rto          uint32 `key:"Rto"`
	Ato          uint32 `key:"Ato"`
	SndMss       uint32 `key:"Snd_mss"`
	RcvMss       uint32 `key:"Rcv_mss"`
	Unacked      uint32 `key:"Unacked"`
	Sacked       uint32 `key:"Sacked"`
	Lost         uint32 `key:"Lost"`
	Retrans      uint32 `key:"Retrans"`
	Fackets      uint32 `key:"Fackets"`
	LastDataSent uint32 `key:"Last_data_sent"`
	LastAckSent  uint32 `key:"Last_ack_sent"`
	LastDataRecv uint32 `key:"Last_data_recv"`
	LastAckRecv  uint32 `key:"Last_ack_recv"`
	Pmtu         uint32 `key:"Pmtu"`
	RcvSsthresh  uint32 `key:"Rcv_ssthresh"`
	Rtt          uint32 `key:"Rtt"`
	Rttvar       uint32 `key:"Rttvar"`
	SndSsthresh  uint32 `key:"Snd_ssthresh"`
	SndCwnd      uint32 `key:"Snd_cwnd"`
	Advmss       uint32 `key:"Advmss"`
	Reordering   uint32 `key:"Reordering"`
	RcvRtt       uint32 `key:"Rcv_rtt"`
	RcvSpace     uint32 `key:"Rcv_space"`
	TotalRetrans uint32 `key:"Total_retrans"`

	HTTPStatusCode int

	Connect  int64
	Resolve  int64
	Download int64
}

type client struct {
	conn net.Conn
	req  *request

	stats
}

func newClient(req *request) *client {
	return &client{req: req}
}

func (c *client) connect() error {
	var err error

	addr, err := c.getAddr()
	if err != nil {
		return err
	}

	t := time.Now()
	d := net.Dialer{LocalAddr: getSrcAddr(c.req.srcAddr)}
	ctx, cancel := context.WithTimeout(context.Background(), c.req.timeout)
	defer cancel()
	c.conn, err = d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}

	c.stats.Connect = time.Since(t).Microseconds()

	return nil
}

func (c *client) dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return c.conn, nil
}

func (c *client) dialTLSContext(ctx context.Context, network, addr string) (net.Conn, error) {
	config := tls.Config{InsecureSkipVerify: c.req.insecure, ServerName: c.serverName()}
	tlsConn := tls.Client(c.conn, &config)

	err := tlsConn.Handshake()

	return tlsConn, err
}

func (c *client) getHostPort() (string, string, error) {
	var host string

	if c.req.urlSchema.Host != "" {
		host = c.req.urlSchema.Host
	} else {
		host = c.req.target
	}

	host, port, err := net.SplitHostPort(host)
	if e, ok := err.(*net.AddrError); ok && e.Err == "missing port in address" {
		if c.req.urlSchema.Host != "" {
			host = c.req.urlSchema.Host
		} else {
			host = c.req.target
		}

		if c.req.urlSchema.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	} else if err != nil {
		return "", "", err
	}

	return host, port, nil
}

func (c *client) getAddr() (string, error) {
	host, port, err := c.getHostPort()
	if err != nil {
		return "", err
	}

	t := time.Now()
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}
	c.stats.Resolve = time.Since(t).Microseconds()

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
		DialContext:       c.dialContext,
		DialTLSContext:    c.dialTLSContext,
		ForceAttemptHTTP2: c.req.http2,
	}

	httpClient := &http.Client{
		Transport:     tr,
		CheckRedirect: c.noRedirect,
	}
	resp, err := httpClient.Get(c.req.target)
	if err != nil {
		return err
	}

	t := time.Now()
	io.Copy(ioutil.Discard, resp.Body)
	c.stats.Download = time.Since(t).Microseconds()

	c.stats.HTTPStatusCode = resp.StatusCode

	resp.Body.Close()

	return nil
}

func (c *client) noRedirect(req *http.Request, via []*http.Request) error {
	//log.Printf("%#v", via[len(via)-1].URL.Host)
	// req.URL.Host == via[len(via)-1].URL.Host
	return fmt.Errorf("%s has been redirected", c.req.target)
}

func (c *client) serverName() string {
	var hostPort string

	if c.req.serverName != "" {
		return c.req.serverName
	}

	if c.req.urlSchema.Host != "" {
		hostPort = c.req.urlSchema.Host
	} else {
		hostPort = c.req.target
	}

	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort
	}

	return host
}

func getSrcAddr(src string) net.Addr {
	if src == "" {
		return nil
	}

	ip := net.ParseIP(src)

	return &net.TCPAddr{IP: ip, Port: 0, Zone: ""}
}

func (c *client) getTCPInfo() error {
	tcpConn := c.conn.(*net.TCPConn)
	if tcpConn == nil {
		return fmt.Errorf("tcp conn is nil")
	}

	file, err := tcpConn.File()
	if err != nil {
		return err
	}
	defer file.Close()

	fd := file.Fd()
	tcpInfo := syscall.TCPInfo{}
	size := uint32(syscall.SizeofTCPInfo)

	_, _, e := syscall.Syscall6(syscall.SYS_GETSOCKOPT, fd, syscall.SOL_TCP, syscall.TCP_INFO,
		uintptr(unsafe.Pointer(&tcpInfo)), uintptr(unsafe.Pointer(&size)), 0)
	if e != 0 {
		return fmt.Errorf("syscall err number=%d", e)
	}

	src := reflect.ValueOf(&tcpInfo).Elem()
	dst := reflect.ValueOf(&c.stats).Elem()

	for i := 0; i < dst.NumField(); i++ {
		if dst.Type().Field(i).Tag.Get("key") == "" {
			continue
		}

		from := src.FieldByName(dst.Type().Field(i).Tag.Get("key")).Addr().Interface()
		to := dst.FieldByName(dst.Type().Field(i).Name).Addr().Interface()
		reflect.ValueOf(to).Elem().Set(reflect.ValueOf(from).Elem())
	}

	return nil
}

func (c *client) reset() {
	s := reflect.ValueOf(&c.stats).Elem()
	s.Set(reflect.Zero(s.Type()))
}
