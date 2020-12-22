package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// stats represents the metrics including socket
// statistics, TCP connect, DNS, TLS, HTTP and errors.
type stats struct {
	State         uint8   `name:"tcpinfo_state" help:"TCP state"`
	CaState       uint8   `name:"tcpinfo_ca_state" help:"state of congestion avoidance"`
	Retransmits   uint8   `name:"tcpinfo_retransmits" help:"number of retranmissions on timeout invoked"`
	Probes        uint8   `name:"tcpinfo_probes" help:"consecutive zero window probes that have gone unanswered"`
	Backoff       uint8   `name:"tcpinfo_backoff" help:"used for exponential backoff re-transmission"`
	Options       uint8   `name:"tcpinfo_options" help:"number of requesting options"`
	pad           [2]byte `unexported:"true"`
	Rto           uint32  `name:"tcpinfo_rto" help:"tcp re-transmission timeout value, the unit is microsecond"`
	Ato           uint32  `name:"tcpinfo_ato" help:"ack timeout, unit is microsecond"`
	SndMss        uint32  `name:"tcpinfo_snd_mss" help:"current maximum segment size"`
	RcvMss        uint32  `name:"tcpinfo_rcv_mss" help:"maximum observed segment size from the remote host"`
	Unacked       uint32  `name:"tcpinfo_unacked" help:"number of unack'd segments"`
	Sacked        uint32  `name:"tcpinfo_sacked" help:"scoreboard segment marked SACKED by sack blocks accounting for the pipe algorithm"`
	Lost          uint32  `name:"tcpinfo_lost" help:"scoreboard segments marked lost by loss detection heuristics accounting for the pipe algorithm"`
	Retrans       uint32  `name:"tcpinfo_retrans" help:"how many times the retran occurs"`
	Fackets       uint32  `name:"tcpinfo_fackets" help:""`
	LastDataSent  uint32  `name:"tcpinfo_last_data_sent" help:"time since last data segment was sent"`
	LastAckSent   uint32  `name:"tcpinfo_last_ack_sent" help:"how long time since the last ack sent"`
	LastDataRecv  uint32  `name:"tcpinfo_last_data_recv" help:"time since last data segment was received"`
	LastAckRecv   uint32  `name:"tcpinfo_last_ack_recv" help:"how long time since the last ack received"`
	Pmtu          uint32  `name:"tcpinfo_path_mtu" help:"path MTU"`
	RcvSsthresh   uint32  `name:"tcpinfo_rev_ss_thresh" help:"tcp congestion window slow start threshold"`
	Rtt           uint32  `name:"tcpinfo_rtt" help:"smoothed round trip time"`
	Rttvar        uint32  `name:"tcpinfo_rtt_var" help:"RTT variance"`
	SndSsthresh   uint32  `name:"tcpinfo_snd_ss_thresh" help:"slow start threshold"`
	SndCwnd       uint32  `name:"tcpinfo_snd_cwnd" help:"congestion window size"`
	Advmss        uint32  `name:"tcpinfo_adv_mss" help:"advertised maximum segment size"`
	Reordering    uint32  `name:"tcpinfo_reordering" help:"number of reordered segments allowed"`
	RcvRtt        uint32  `name:"tcpinfo_rcv_rtt" help:"receiver side RTT estimate"`
	RcvSpace      uint32  `name:"tcpinfo_rcv_space" help:"space reserved for the receive queue"`
	TotalRetrans  uint32  `name:"tcpinfo_total_retrans" help:"total number of segments containing retransmitted data"`
	PacingRate    uint64  `name:"tcpinfo_pacing_rate" help:"the pacing rate"`
	maxPacingRate uint64  `name:"tcpinfo_max_pacing_rate" help:"" unexported:"true"`
	BytesAcked    uint64  `name:"tcpinfo_bytes_acked" help:"bytes acked"`
	BytesReceived uint64  `name:"tcpinfo_bytes_received" help:"bytes received"`
	SegsOut       uint32  `name:"tcpinfo_segs_out" help:"segments sent out"`
	SegsIn        uint32  `name:"tcpinfo_segs_in" help:"segments received"`
	NotsentBytes  uint32  `name:"tcpinfo_notsent_bytes" help:""`
	MinRtt        uint32  `name:"tcpinfo_min_rtt" help:""`
	DataSegsIn    uint32  `name:"tcpinfo_data_segs_in" help:"RFC4898 tcpEStatsDataSegsIn"`
	DataSegsOut   uint32  `name:"tcpinfo_data_segs_out" help:"RFC4898 tcpEStatsDataSegsOut"`
	DeliveryRate  uint64  `name:"tcpinfo_delivery_rate" help:""`
	BusyTime      uint64  `name:"tcpinfo_busy_time" help:"time (usec) busy sending data"`
	RwndLimited   uint64  `name:"tcpinfo_rwnd_limited" help:"time (usec) limited by receive window"`
	SndbufLimited uint64  `name:"tcpinfo_sndbuf_limited" help:"time (usec) limited by send buffer"`
	Delivered     uint32  `name:"tcpinfo_delivered" help:""`
	DeliveredCe   uint32  `name:"tcpinfo_delivered_ce" help:""`
	BytesSent     uint64  `name:"tcpinfo_bytes_sent" help:""`
	BytesRetrans  uint64  `name:"tcpinfo_bytes_retrans" help:"RFC4898 tcpEStatsPerfOctetsRetrans"`
	DsackDups     uint32  `name:"tcpinfo_dsack_dups" help:"RFC4898 tcpEStatsStackDSACKDups"`
	ReordSeen     uint32  `name:"tcpinfo_reord_seen" help:"reordering events seen"`
	RcvOoopack    uint32  `name:"tcpinfo_rcv_ooopack" help:"out-of-order packets received"`
	SndWnd        uint32  `name:"tcpinfo_snd_wnd" help:""`

	TCPCongesAlg string `help:"TCP network congestion-avoidance algorithm"`

	HTTPStatusCode int   `name:"http_status_code" help:"HTTP 1xx-5xx status code"`
	HTTPRcvdBytes  int64 `name:"http_rcvd_bytes" help:"HTTP bytes received"`
	HTTPRequest    int64 `name:"http_request" help:"HTTP request, the unit is microsecond"`
	HTTPResponse   int64 `name:"http_response" help:"HTTP response, the unit is microsecond"`

	DNSResolve   int64 `name:"dns_resolve" help:"domain lookup, the unit is microsecond"`
	TCPConnect   int64 `name:"tcp_connect" help:"TCP connect, the unit is microsecond"`
	TLSHandshake int64 `name:"tls_handshake" help:"TLS handshake, the unit is microsecond"`

	TCPConnectError int64 `name:"tcp_connect_error" help:"total TCP connect error" kind:"counter"`
	DNSResolveError int64 `name:"dns_resolve_error" help:"total DNS resolve error" kind:"counter"`
}

// client represents a proble client to specific target
type client struct {
	target    string
	addr      string
	timestamp int64
	urlSchema *url.URL

	conn net.Conn
	req  *request

	subCh []chan *stats
	mu    *sync.Mutex

	stats
}

func newClient(req *request, target string) *client {
	urlSchema, err := url.Parse(target)
	if err != nil {
		urlSchema = &url.URL{}
	}

	c := &client{
		target:    target,
		urlSchema: urlSchema,
		req:       req,
	}

	if req.grpc {
		c.mu = &sync.Mutex{}
	}

	return c
}

func (c *client) connect(ctx context.Context) error {
	var err error

	c.timestamp = time.Now().Unix()

	addr, err := c.getAddr()
	if err != nil {
		return err
	}

	c.addr = addr

	d := net.Dialer{
		LocalAddr: getSrcAddr(c.req.srcAddr),
		Control:   c.control,
	}
	ctx, cancel := context.WithTimeout(ctx, c.req.timeout)
	defer cancel()

	t := time.Now()
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

	t := time.Now()
	err := tlsConn.Handshake()
	c.stats.TLSHandshake = time.Since(t).Microseconds()

	return tlsConn, err
}

func (c *client) control(network string, address string, conn syscall.RawConn) error {
	return conn.Control(func(fd uintptr) {

		setSocketOptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_PRIORITY, c.req.soPriority, false)
		setSocketOptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, c.req.soSndBuf, false)
		setSocketOptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, c.req.soRcvBuf, false)
		setSocketOptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_NODELAY, boolToInt(!c.req.soTCPNoDelay), true)
		setSocketOptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_QUICKACK, boolToInt(!c.req.soTCPQuickACK), true)
		setSocketOptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_MAXSEG, c.req.soMaxSegSize, false)

		if c.isIPv4() {
			setSocketOptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TOS, c.req.soIPTOS, false)
			setSocketOptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, c.req.soIPTTL, false)
		} else {
			setSocketOptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_UNICAST_HOPS, c.req.soIPTTL, false)
			setSocketOptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_TCLASS, c.req.soIPTOS, false)
		}

		err := syscall.SetsockoptString(int(fd), syscall.IPPROTO_TCP, syscall.TCP_CONGESTION, c.req.soCongestion)
		if c.req.soCongestion != "" && err != nil {
			log.Fatal(os.NewSyscallError("congestion-avoidance algorithm error", err))
		}
	})
}

func setSocketOptInt(fd int, level int, opt int, value int, zeroExc bool) {
	if (value == 0 && !zeroExc) || (value == 1 && zeroExc) {
		return
	}

	err := syscall.SetsockoptInt(fd, level, opt, value)
	if err != nil {
		log.Println(os.NewSyscallError("setsockopt", err))
	}
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

	if ok := isIPAddr(host); ok {
		return net.JoinHostPort(host, port), nil
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
				return net.JoinHostPort(addr, port), nil
			}

			if c.req.ipv4 {
				continue
			}
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

func (c *client) isIPv4() bool {
	return net.ParseIP(c.addr).To4() != nil
}

func (c *client) httpGet() error {
	tr := &http.Transport{
		DialContext:       c.dialContext,
		DialTLSContext:    c.dialTLSContext,
		ForceAttemptHTTP2: c.req.http2,
	}

	httpClient := &http.Client{
		Timeout:       c.req.timeoutHTTP,
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

func (c *client) probe(ctx context.Context) {
	counter := -1
	wait := c.getInterval(ctx)
	for counter < c.req.count-1 || c.req.count == 0 {
		counter++

		if counter != 0 {
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return
			}
		}

		err := c.connect(ctx)
		if err != nil {
			if ctx.Err() == nil {
				log.Println(err)
			}
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

		if c.req.grpc {
			c.publish()
		}

		c.printer(counter)

		c.close()
	}
}

func (c *client) publish() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, ch := range c.subCh {
		select {
		case ch <- &c.stats:
		default:
		}
	}
}

func (c *client) subscribe(ch chan *stats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subCh = append(c.subCh, ch)
}

func (c *client) unsubscribe(ch chan *stats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, sCh := range c.subCh {
		if ch == sCh {
			c.subCh = append(c.subCh[:i], c.subCh[i+1:]...)
		}
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
		return errors.New("tcp conn is nil")
	}

	file, err := tcpConn.File()
	if err != nil {
		return err
	}
	defer file.Close()

	fd := file.Fd()
	size := uint32(232)

	_, _, e := syscall.Syscall6(syscall.SYS_GETSOCKOPT, fd, syscall.SOL_TCP, syscall.TCP_INFO,
		uintptr(unsafe.Pointer(&c.stats)), uintptr(unsafe.Pointer(&size)), 0)
	if e != 0 {
		return fmt.Errorf("syscall err number=%d", e)
	}

	ca := make([]byte, 10)
	size = uint32(len(ca))

	_, _, e = syscall.Syscall6(syscall.SYS_GETSOCKOPT, fd, syscall.IPPROTO_TCP, syscall.TCP_CONGESTION,
		uintptr(unsafe.Pointer(&ca[0])), uintptr(unsafe.Pointer(&size)), 0)
	if e != 0 {
		return fmt.Errorf("syscall err number=%d", e)
	}

	c.stats.TCPCongesAlg = string(bytes.Trim(ca, "\x00"))

	return nil
}

func (c *client) getInterval(ctx context.Context) time.Duration {
	if v := ctx.Value(intervalKey); v != nil {
		d, err := time.ParseDuration(v.(string))
		if err != nil || d == 0 {
			return c.req.interval
		}

		return d
	}

	return c.req.interval
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func isIPAddr(host string) bool {
	addr := net.ParseIP(host)
	if addr != nil {
		return true
	}

	return false
}
