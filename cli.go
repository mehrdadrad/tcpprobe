package main

import (
	"net/url"
	"time"

	cli "github.com/urfave/cli/v2"
)

type request struct {
	target    string
	urlSchema *url.URL
	count     int
	ipv6      bool

	timeout time.Duration
	wait    time.Duration
}

func getCli(args []string) (*request, error) {
	r := &request{}
	flags := []cli.Flag{
		&cli.BoolFlag{Name: "ipv6", Aliases: []string{"6"}},
		&cli.IntFlag{Name: "count", Aliases: []string{"c"}, Value: 1},
		&cli.DurationFlag{Name: "timeout", Aliases: []string{"t"}, Value: time.Second},
		&cli.DurationFlag{Name: "wait", Aliases: []string{"w"}, Value: time.Second},
	}

	app := &cli.App{
		Flags: flags,
		Action: func(c *cli.Context) error {
			if c.NArg() > 0 {
				r.target = c.Args().Get(c.NArg() - 1)
				r.urlSchema, _ = url.Parse(r.target)
			}

			r.ipv6 = c.Bool("6")
			r.count = c.Int("c")
			r.wait = c.Duration("wait")
			r.timeout = c.Duration("timeout")

			return nil
		},
	}

	err := app.Run(args)

	return r, err
}
