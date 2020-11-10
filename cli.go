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

type request struct {
	count        int
	ipv4         bool
	ipv6         bool
	http2        bool
	json         bool
	quiet        bool
	insecure     bool
	promDisabled bool
	promAddr     string
	serverName   string
	srcAddr      string
	filter       string

	soIPTOS      int
	soIPTTL      int
	soPriority   int
	soMaxSegSize int
	soTCPNoDelay bool

	timeout time.Duration
	wait    time.Duration
}

func getCli(args []string) (*request, []string, error) {
	var (
		r       = &request{}
		targets []string
	)

	flags := []cli.Flag{
		&cli.BoolFlag{Name: "ipv6", Aliases: []string{"6"}, Usage: "connect only to IPv6 address"},
		&cli.BoolFlag{Name: "ipv4", Aliases: []string{"4"}, Usage: "connect only to IPv4 address"},
		&cli.IntFlag{Name: "count", Aliases: []string{"c"}, Value: 1, Usage: "stop after sending count requests [0 is unlimited]"},
		&cli.BoolFlag{Name: "http2", Usage: "force to use HTTP version 2"},
		&cli.BoolFlag{Name: "json", Usage: "print in json format"},
		&cli.BoolFlag{Name: "prom-disabled", Usage: "disable prometheus"},
		&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}, Usage: "turn off tcpprobe output"},
		&cli.BoolFlag{Name: "insecure", Aliases: []string{"i"}, Usage: "don't validate the server's certificate"},
		&cli.StringFlag{Name: "server-name", Aliases: []string{"n"}, Usage: "server name is used to verify the hostname (TLS)"},
		&cli.StringFlag{Name: "source-addr", Aliases: []string{"S"}, Usage: "source address in outgoing request"},
		&cli.StringFlag{Name: "prom-addr", Aliases: []string{"p"}, Value: ":8081", Usage: "specify prometheus exporter IP and port"},
		&cli.StringFlag{Name: "filter", Aliases: []string{"f"}, Usage: "given metric(s) with semicolon delimited"},
		&cli.DurationFlag{Name: "timeout", Aliases: []string{"t"}, Value: time.Second, Usage: "specify a timeout for dialing to targets"},
		&cli.DurationFlag{Name: "wait", Aliases: []string{"w"}, Value: time.Second, Usage: "time to wait after each request"},
		&cli.IntFlag{Name: "tos", Aliases: []string{"z"}, DefaultText: "depends on the OS", Usage: "set the IP type of service"},
		&cli.IntFlag{Name: "ttl", Aliases: []string{"m"}, DefaultText: "depends on the OS", Usage: "set the IP time to live"},
		&cli.IntFlag{Name: "socket-priority", Aliases: []string{"r"}, DefaultText: "depends on the OS", Usage: "set queuing discipline"},
		&cli.IntFlag{Name: "mss", Aliases: []string{"M"}, DefaultText: "depends on the OS", Usage: "TCP max segment size"},
		&cli.BoolFlag{Name: "tcp-nodelay-disabled", Aliases: []string{"o"}, Usage: "disable Nagle's algorithm"},
		&cli.BoolFlag{Name: "metrics", Usage: "show metric's descriptions"},
	}

	app := &cli.App{
		Flags: flags,
		Action: func(c *cli.Context) error {
			r = &request{
				ipv4:         c.Bool("ipv4"),
				ipv6:         c.Bool("ipv6"),
				http2:        c.Bool("http2"),
				json:         c.Bool("json"),
				quiet:        c.Bool("quiet"),
				insecure:     c.Bool("insecure"),
				promDisabled: c.Bool("prom-disabled"),
				promAddr:     c.String("prom-addr"),
				serverName:   c.String("server-name"),
				srcAddr:      c.String("source-addr"),
				filter:       c.String("filter"),
				count:        c.Int("count"),

				soIPTOS:      c.Int("tos"),
				soIPTTL:      c.Int("ttl"),
				soPriority:   c.Int("socket-priority"),
				soMaxSegSize: c.Int("mss"),
				soTCPNoDelay: c.Bool("tcp-nodelay-disabled"),

				wait:    c.Duration("wait"),
				timeout: c.Duration("timeout"),
			}

			if c.Bool("metrics") {
				fmt.Println("metrics:")
				v := reflect.ValueOf(&stats{}).Elem()
				for i := 0; i < v.NumField(); i++ {
					fmt.Printf("%s %s\n", v.Type().Field(i).Name, v.Type().Field(i).Tag.Get("help"))
				}

				return nil
			}

			targets = c.Args().Slice()
			if len(targets) < 1 {
				cli.ShowAppHelp(c)
				return errors.New("configuration not specified")
			}

			return nil
		},
	}

	cli.AppHelpTemplate = `usage: {{.HelpName}} options target(s)

options:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{if .Copyright }}
COPYRIGHT:
   {{.Copyright}}
   {{end}}{{if .Version}}
VERSION:
   {{.Version}}
   {{end}}
`

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
