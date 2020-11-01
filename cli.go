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

	timeout time.Duration
	wait    time.Duration
}

func getCli(args []string) (*request, []string, error) {
	var (
		r       = &request{}
		targets []string
	)

	flags := []cli.Flag{
		&cli.BoolFlag{Name: "ipv6", Aliases: []string{"6"}},
		&cli.BoolFlag{Name: "ipv4", Aliases: []string{"4"}},
		&cli.BoolFlag{Name: "http2"},
		&cli.BoolFlag{Name: "json"},
		&cli.BoolFlag{Name: "prom-disabled"},
		&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}},
		&cli.BoolFlag{Name: "insecure", Aliases: []string{"i"}},
		&cli.StringFlag{Name: "server-name", Aliases: []string{"n"}},
		&cli.StringFlag{Name: "source-addr", Aliases: []string{"S"}},
		&cli.StringFlag{Name: "prom-addr", Aliases: []string{"p"}, Value: ":8081"},
		&cli.IntFlag{Name: "count", Aliases: []string{"c"}, Value: 1},
		&cli.DurationFlag{Name: "timeout", Aliases: []string{"t"}, Value: time.Second},
		&cli.DurationFlag{Name: "wait", Aliases: []string{"w"}, Value: time.Second},
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
				count:        c.Int("count"),
				wait:         c.Duration("wait"),
				timeout:      c.Duration("timeout"),
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
