package main

import (
	"net/url"
	"time"

	cli "github.com/urfave/cli/v2"
)

type request struct {
	target     string
	urlSchema  *url.URL
	count      int
	ipv6       bool
	http2      bool
	insecure   bool
	serverName string
	srcAddr    string
	promAddr   string

	timeout time.Duration
	wait    time.Duration
}

func getCli(args []string) (*request, error) {
	var r *request

	flags := []cli.Flag{
		&cli.BoolFlag{Name: "ipv6", Aliases: []string{"6"}},
		&cli.BoolFlag{Name: "http2"},
		&cli.BoolFlag{Name: "insecure", Aliases: []string{"i"}},
		&cli.StringFlag{Name: "server-name", Aliases: []string{"n"}},
		&cli.StringFlag{Name: "source-addr", Aliases: []string{"S"}},
		&cli.StringFlag{Name: "prometheus-addr", Aliases: []string{"p"}, Value: ":8080"},
		&cli.IntFlag{Name: "count", Aliases: []string{"c"}, Value: 1},
		&cli.DurationFlag{Name: "timeout", Aliases: []string{"t"}, Value: time.Second},
		&cli.DurationFlag{Name: "wait", Aliases: []string{"w"}, Value: time.Second},
	}

	app := &cli.App{
		Flags: flags,
		Action: func(c *cli.Context) error {
			r = &request{
				ipv6:       c.Bool("ipv6"),
				http2:      c.Bool("http2"),
				count:      c.Int("count"),
				wait:       c.Duration("wait"),
				timeout:    c.Duration("timeout"),
				insecure:   c.Bool("insecure"),
				serverName: c.String("server-name"),
				srcAddr:    c.String("source-addr"),
				promAddr:   c.String("prometheus-addr"),
			}

			if c.NArg() > 0 {
				r.target = c.Args().Get(c.NArg() - 1)
				r.urlSchema, _ = url.Parse(r.target)
			}

			return nil
		},
	}

	err := app.Run(args)

	return r, err
}
