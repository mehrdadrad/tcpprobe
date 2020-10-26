package main

import (
	"log"
	"os"
	"strings"
)

func main() {
	r, err := getCli(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < r.count; i++ {
		c, err := newClient(r)
		if err != nil {
			log.Fatal(err)
		}

		if strings.HasPrefix(c.req.target, "http") {
			if err := c.httpGet(); err != nil {
				log.Println(err)
			}
		}

		log.Printf("%d %#v", i, c)

		c.close()
	}
}
