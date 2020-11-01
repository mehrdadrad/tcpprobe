package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var wg sync.WaitGroup

	req, targets, err := getCli(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
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

	if !req.promDisabled {
		http.Handle("/metrics", promhttp.Handler())
		go http.ListenAndServe(req.promAddr, nil)
	}

	wg.Wait()
}
