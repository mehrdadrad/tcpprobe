package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var wg sync.WaitGroup

	req, targets, err := getCli(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	for _, target := range targets {
		wg.Add(1)
		target := target

		go func() {
			defer wg.Done()

			counter := 0
			c := newClient(req, target)
			c.prometheus()

			for {
				err = c.connect()
				if err != nil {
					log.Println(err)
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
		}()
	}

	time.Sleep(time.Second)

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(req.promAddr, nil)

	wg.Wait()
}
