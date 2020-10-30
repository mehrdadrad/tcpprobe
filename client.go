package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type stats struct {
	State        uint8  `key:"State" name:"tcpinfo_state" help:"TCP State"`
	CaState      uint8  `key:"Ca_state" name:"tcpinfo_ca_state" help:""`
	Retransmits  uint8  `key:"Retransmits" name:"tcpinfo_retransmits" help:""`
	Probes       uint8  `key:"Probes" name:"tcpinfo_probes" help:"consecutive zero window probes that have gone unanswered"`
	Backoff      uint8  `key:"Backoff" name:"tcpinfo_backoff" help:"used for exponential backoff re-transmission"`
	Options      uint8  `key:"Options" name:"tcpinfo_options" help:""`
	Rto          uint32 `key:"Rto" name:"tcpinfo_rto" help:"tcp re-transmission timeout value, the unit is microsecond"`
	Ato          uint32 `key:"Ato" name:"tcpinfo_ato" help:"ack timeout, unit is microsecond"`
	SndMss       uint32 `key:"Snd_mss" name:"tcpinfo_snd_mss" help:"current maximum segment size"`
	RcvMss       uint32 `key:"Rcv_mss" name:"tcpinfo_rcv_mss" help:"maximum observed segment size from the remote host"`
	Unacked      uint32 `key:"Unacked" name:"tcpinfo_unacked" help:""`
	Sacked       uint32 `key:"Sacked" name:"tcpinfo_sacked" help:"scoreboard segment marked SACKED by sack blocks accounting for the pipe algorithm"`
	Lost         uint32 `key:"Lost" name:"tcpinfo_lost" help:"scoreboard segments marked lost by loss detection heuristics accounting for the pipe algorithm"`
	Retrans      uint32 `key:"Retrans" name:"tcpinfo_retrans" help:"how many times the retran occurs"`
	Fackets      uint32 `key:"Fackets" name:"tcpinfo_fackets" help:""`
	LastDataSent uint32 `key:"Last_data_sent" name:"tcpinfo_last_data_sent" help:"time since last data segment was sent"`
	LastAckSent  uint32 `key:"Last_ack_sent" name:"tcpinfo_last_ack_sent" help:"how long time since the last ack sent"`
	LastDataRecv uint32 `key:"Last_data_recv" name:"tcpinfo_last_data_recv" help:"time since last data segment was received"`
	LastAckRecv  uint32 `key:"Last_ack_recv" name:"tcpinfo_last_ack_recv" help:"how long time since the last ack received"`
	Pmtu         uint32 `key:"Pmtu" name:"tcpinfo_path_mtu" help:"path MTU"`
	RcvSsthresh  uint32 `key:"Rcv_ssthresh" name:"tcpinfo_rev_ss_thresh" help:"tcp congestion window slow start threshold"`
	Rtt          uint32 `key:"Rtt" name:"tcpinfo_rtt" help:"smoothed round trip time"`
	Rttvar       uint32 `key:"Rttvar" name:"tcpinfo_rtt_var" help:"RTT variance"`
	SndSsthresh  uint32 `key:"Snd_ssthresh" name:"tcpinfo_snd_ss_thresh" help:"slow start threshold"`
	SndCwnd      uint32 `key:"Snd_cwnd" name:"tcpinfo_snd_cwnd" help:""`
	Advmss       uint32 `key:"Advmss" name:"tcpinfo_adv_mss" help:""`
	Reordering   uint32 `key:"Reordering" name:"tcpinfo_reordering" help:""`
	RcvRtt       uint32 `key:"Rcv_rtt" name:"tcpinfo_rcv_rtt" help:"receiver side RTT estimate"`
	RcvSpace     uint32 `key:"Rcv_space" name:"tcpinfo_rcv_space" help:"space reserved for the receive queue"`
	TotalRetrans uint32 `key:"Total_retrans" name:"tcpinfo_total_retrans" help:"total number of segments containing retransmitted data"`

	HTTPStatusCode int   `name:"http_status_code" help:"HTTP 1xx-5xx status code"`
	HTTPRcvdBytes  int64 `name:"http_rcvd_bytes" help:""`
	HTTPRequest    int64 `name:"http_request" help:""`
	HTTPResponse   int64 `name:"http_response" help:""`

	DNSResolve int64 `name:"dns_resolve" help:""`
	TCPConnect int64 `name:"tcp_connect" help:""`

	TCPConnectError int64 `name:"tcp_connect_error" help:""`
	DNSResolveError int64 `name:"dns_resolve_error" help:""`
}

type client struct {
	target    string
	addr      string
	timestamp int64
	urlSchema *url.URL

	conn net.Conn
	req  *request

	stats
}

func newClient(req *request, target string) *client {
	urlSchema, _ := url.Parse(target)
	return &client{
		target:    target,
		urlSchema: urlSchema,
		req:       req,
	}
}

func (c *client) connect() error {
	var err error

	c.timestamp = time.Now().Unix()

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
		c.stats.TCPConnectError++
		return err
	}

	c.stats.TCPConnect = time.Since(t).Microseconds()

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

	if c.urlSchema.Host != "" {
		host = c.urlSchema.Host
	} else {
		host = c.target
	}

	host, port, err := net.SplitHostPort(host)
	if e, ok := err.(*net.AddrError); ok && e.Err == "missing port in address" {
		if c.urlSchema.Host != "" {
			host = c.urlSchema.Host
		} else {
			host = c.target
		}

		if c.urlSchema.Scheme == "https" {
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
		c.stats.DNSResolveError++
		return "", err
	}
	c.stats.DNSResolve = time.Since(t).Microseconds()

	for _, addr := range addrs {
		// IPv4 requested
		if !c.req.ipv6 {
			if net.ParseIP(addr).To4() != nil {
				c.addr = addr
				return net.JoinHostPort(addr, port), nil
			}
			continue
		}

		// IPv6 requested
		if net.ParseIP(addr).To4() == nil {
			c.addr = addr
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
	t := time.Now()
	resp, err := httpClient.Get(c.target)
	if err != nil {
		return err
	}
	c.stats.HTTPRequest = time.Since(t).Microseconds()

	t = time.Now()
	written, err := io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		return err
	}
	c.stats.HTTPResponse = time.Since(t).Microseconds()

	c.stats.HTTPStatusCode = resp.StatusCode
	c.stats.HTTPRcvdBytes = written

	resp.Body.Close()

	return nil
}

func (c *client) noRedirect(req *http.Request, via []*http.Request) error {
	//log.Printf("%#v", via[len(via)-1].URL.Host)
	// req.URL.Host == via[len(via)-1].URL.Host
	return fmt.Errorf("%s has been redirected", c.target)
}

func (c *client) serverName() string {
	var hostPort string

	if c.req.serverName != "" {
		return c.req.serverName
	}

	if c.urlSchema.Host != "" {
		hostPort = c.urlSchema.Host
	} else {
		hostPort = c.target
	}

	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort
	}

	return host
}

func (c *client) probe() {
	counter := 0

	for {
		err := c.connect()
		if err != nil {
			log.Println(err)
			time.Sleep(c.req.wait)
			continue
		}

		if strings.HasPrefix(c.target, "http") {
			if err := c.httpGet(); err != nil {
				log.Println(err)
			}
		}

		if err = c.getTCPInfo(); err != nil {
			log.Println(err)
		}

		c.printer()

		c.close()
		counter++

		if counter >= c.req.count && c.req.count != 0 {
			break
		}

		time.Sleep(c.req.wait)
	}
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
