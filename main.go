package main

import (
	"log"
	"net/http"
	"os"
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

	wg.Add(len(targets))
	for _, target := range targets {
		target := target

		go func() {
			defer wg.Done()
			c := newClient(req, target)
			c.prometheus()
			c.probe()
		}()
	}

	time.Sleep(time.Second)

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(req.promAddr, nil)

	wg.Wait()
}
