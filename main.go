package main

import (
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	var (
		counter int
	)
	r, err := getCli(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	c := newClient(r)
	c.prometheus()

	for {
		c.reset()

		err = c.connect()
		if err != nil {
			log.Fatal(err)
		}

		if strings.HasPrefix(c.req.target, "http") {
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
