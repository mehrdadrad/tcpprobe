package main

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/template"
	"time"

	cli "github.com/urfave/cli/v2"
)

// request represents tcpprobe request's parameters
type request struct {
	count        int
	ipv4         bool
	ipv6         bool
	http2        bool
	k8s          bool
	json         bool
	jsonPretty   bool
	grpc         bool
	quiet        bool
	insecure     bool
	promDisabled bool
	grpcAddr     string
	namespace    string
	promAddr     string
	serverName   string
	srcAddr      string
	filter       string
	config       string

	soIPTOS       int
	soIPTTL       int
	soPriority    int
	soMaxSegSize  int
	soSndBuf      int
	soRcvBuf      int
	soCongestion  string
	soTCPNoDelay  bool
	soTCPQuickACK bool

	timeout     time.Duration
	timeoutHTTP time.Duration
	interval    time.Duration

	cmd *cmdReq

	checkUpdate bool
}

// cmdReq represents grpc commands: add and delete
type cmdReq struct {
	cmd      string
	insecure bool
	addr     string
	labels   string
	interval string
	args     []string
}

func getCli(args []string) (*request, []string, error) {
	var (
		r       = &request{}
		targets []string
	)

	cmdFlags := []cli.Flag{
		&cli.StringFlag{Name: "interval", Aliases: []string{"i"}, Value: "5s", Usage: "time to wait after each request"},
		&cli.StringFlag{Name: "addr", Aliases: []string{"d"}, Value: "localhost:8082", Usage: "tcpprobe grpc server address"},
		&cli.StringFlag{Name: "labels", Aliases: []string{"l"}, Usage: "set labels"},
		&cli.BoolFlag{Name: "insecure", Value: true, Usage: "don't validate the server's certificate"},
	}

	flags := []cli.Flag{
		&cli.BoolFlag{Name: "ipv6", Aliases: []string{"6"}, Usage: "connect only to IPv6 address"},
		&cli.BoolFlag{Name: "ipv4", Aliases: []string{"4"}, Usage: "connect only to IPv4 address"},
		&cli.IntFlag{Name: "count", Aliases: []string{"c"}, Value: 0, Usage: "stop after sending count requests [0 is unlimited]"},
		&cli.BoolFlag{Name: "http2", Usage: "force to use HTTP version 2"},
		&cli.BoolFlag{Name: "prom-disabled", Usage: "disable prometheus"},
		&cli.BoolFlag{Name: "insecure", Usage: "don't validate the server's certificate"},
		&cli.StringFlag{Name: "server-name", Aliases: []string{"n"}, Usage: "server name is used to verify the hostname (TLS)"},
		&cli.StringFlag{Name: "source-addr", Aliases: []string{"S"}, Usage: "source address in outgoing request"},
		&cli.StringFlag{Name: "prom-addr", Aliases: []string{"p"}, Value: ":8081", Usage: "specify prometheus exporter IP and port"},
		&cli.StringFlag{Name: "filter", Aliases: []string{"f"}, Usage: "given metric(s) with semicolon delimited"},
		&cli.DurationFlag{Name: "timeout", Aliases: []string{"t"}, Value: 5 * time.Second, Usage: "specify a timeout for dialing to targets"},
		&cli.DurationFlag{Name: "http-timeout", Aliases: []string{}, Value: 30 * time.Second, Usage: "specify a timeout for HTTP"},
		&cli.DurationFlag{Name: "interval", Aliases: []string{"i"}, Value: time.Second, Usage: "time to wait after each request"},
		&cli.IntFlag{Name: "tos", Aliases: []string{"z"}, DefaultText: "depends on the OS", Usage: "set the IP type of service or traffic class"},
		&cli.IntFlag{Name: "ttl", Aliases: []string{"m"}, DefaultText: "depends on the OS", Usage: "set the IP time to live or hop limit"},
		&cli.IntFlag{Name: "socket-priority", Aliases: []string{"r"}, DefaultText: "depends on the OS", Usage: "set queuing discipline"},
		&cli.IntFlag{Name: "mss", Aliases: []string{"M"}, DefaultText: "depends on the OS", Usage: "TCP maximum segment size"},
		&cli.StringFlag{Name: "congestion-alg", Aliases: []string{}, DefaultText: "depends on the OS", Usage: "TCP congestion control algorithm"},
		&cli.IntFlag{Name: "send-buffer", Aliases: []string{}, DefaultText: "depends on the OS", Usage: "maximum socket send buffer in bytes"},
		&cli.IntFlag{Name: "rcvd-buffer", Aliases: []string{}, DefaultText: "depends on the OS", Usage: "maximum socket receive buffer in bytes"},
		&cli.BoolFlag{Name: "tcp-nodelay-disabled", Aliases: []string{"o"}, Usage: "disable Nagle's algorithm"},
		&cli.BoolFlag{Name: "tcp-quickack-disabled", Aliases: []string{"k"}, Usage: "disable quickack mode"},
		&cli.BoolFlag{Name: "k8s", Usage: "enable k8s"},
		&cli.StringFlag{Name: "namespace", Value: "default", Usage: "kubernetes namespace"},
		&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}, Usage: "turn off tcpprobe output"},
		&cli.BoolFlag{Name: "json", Usage: "print in json format"},
		&cli.BoolFlag{Name: "json-pretty", Usage: "pretty print in json format"},
		&cli.BoolFlag{Name: "grpc", Usage: "enable grpc"},
		&cli.StringFlag{Name: "grpc-addr", Aliases: []string{"g"}, Value: ":8082", Usage: "specify grpc server IP and port"},
		&cli.BoolFlag{Name: "metrics", Usage: "show metrics descriptions"},
		&cli.StringFlag{Name: "config", Usage: "yaml config file"},
		&cli.BoolFlag{Name: "check-update", Usage: "check for update"},
	}

	app := &cli.App{
		Version: version,
		Flags:   flags,
		Commands: []*cli.Command{
			{
				Name:  "add",
				Usage: "add target through grpc",
				Flags: cmdFlags,
				Action: func(c *cli.Context) error {
					r.cmd = &cmdReq{
						cmd:      "add",
						insecure: c.Bool("insecure"),
						addr:     c.String("addr"),
						interval: c.String("interval"),
						labels:   c.String("labels"),
						args:     c.Args().Slice(),
					}

					targets = c.Args().Slice()
					if len(targets) < 1 {
						cli.ShowCommandHelp(c, "add")
						return errors.New("configuration not specified")
					}

					return nil
				},
			},
			{
				Name:  "del",
				Usage: "delete target through grpc",
				Flags: cmdFlags,
				Action: func(c *cli.Context) error {
					r.cmd = &cmdReq{
						cmd:      "del",
						insecure: c.Bool("insecure"),
						addr:     c.String("addr"),
						interval: c.String("interval"),
						labels:   c.String("labels"),
						args:     c.Args().Slice(),
					}

					targets = c.Args().Slice()
					if len(targets) < 1 {
						cli.ShowCommandHelp(c, "del")
						return errors.New("configuration not specified")
					}

					return nil
				},
			},
		},
		Action: func(c *cli.Context) error {
			r = &request{
				ipv4:         c.Bool("ipv4"),
				ipv6:         c.Bool("ipv6"),
				http2:        c.Bool("http2"),
				k8s:          c.Bool("k8s"),
				json:         c.Bool("json"),
				jsonPretty:   c.Bool("json-pretty"),
				grpc:         c.Bool("grpc"),
				quiet:        c.Bool("quiet"),
				insecure:     c.Bool("insecure"),
				promDisabled: c.Bool("prom-disabled"),
				namespace:    c.String("namespace"),
				promAddr:     c.String("prom-addr"),
				grpcAddr:     c.String("grpc-addr"),
				serverName:   c.String("server-name"),
				srcAddr:      c.String("source-addr"),
				filter:       c.String("filter"),
				config:       c.String("config"),
				count:        c.Int("count"),

				soIPTOS:      c.Int("tos"),
				soIPTTL:      c.Int("ttl"),
				soPriority:   c.Int("socket-priority"),
				soMaxSegSize: c.Int("mss"),
				soSndBuf:     c.Int("send-buffer"),
				soRcvBuf:     c.Int("rcvd-buffer"),
				soCongestion: c.String("congestion-alg"),
				soTCPNoDelay: c.Bool("tcp-nodelay-disabled"),

				interval:    c.Duration("interval"),
				timeout:     c.Duration("timeout"),
				timeoutHTTP: c.Duration("http-timeout"),
			}

			if c.Bool("metrics") {
				fmt.Println("metrics:")
				v := reflect.ValueOf(&stats{}).Elem()
				for i := 0; i < v.NumField(); i++ {
					f := v.Type().Field(i)
					if f.Tag.Get("unexported") == "true" {
						continue
					}
					fmt.Printf("%s %s\n", f.Name, f.Tag.Get("help"))
				}

				return nil
			}

			if c.Bool("check-update") {
				ok, newVersion := checkUpdate(tpReleaseURL)
				if ok {
					fmt.Printf("the new version: v%s available\n", newVersion)
				} else {
					fmt.Println("there is currently no update available")
				}
				return nil
			}

			targets = c.Args().Slice()
			if len(targets) < 1 && len(r.config) < 1 && !r.k8s && !r.grpc {
				cli.ShowAppHelp(c)
				return errors.New("configuration not specified")
			}

			return nil
		},
	}

	cli.AppHelpTemplate = `usage: {{.HelpName}} options target(s)

options:
   {{range .VisibleFlags}}{{.}}
   {{end}}
examples:   
   tcpprobe -json -c 0 https://www.google.com
   tcpprobe -filter "Rtt;TCPConnect" https://www.yahoo.com
   tcpprobe smtp.gmail.com:587

for more information: https://github.com/mehrdadrad/tcpprobe/wiki   
`

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("tcpprobe version %s\n", c.App.Version)
		cli.OsExiter(0)
	}

	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		funcMap := template.FuncMap{
			"join": strings.Join,
		}
		t := template.Must(template.New("help").Funcs(funcMap).Parse(templ))
		t.Execute(w, data)
	}

	err := app.Run(args)

	return r, targets, err
}
