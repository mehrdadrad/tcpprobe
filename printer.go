package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"reflect"
	"strings"
)

func (c *client) printer(counter int) {
	if c.req.quiet {
		return
	}

	switch {
	case c.req.json:
		c.printJSON(counter)
	case c.req.jsonPretty:
		c.printJSONPretty(counter)
	default:
		c.printText(counter)
	}
}

func (c *client) printText(counter int) {
	v := reflect.ValueOf(c.stats)
	filter := strings.ToLower(c.req.filter)

	ip, _, _ := net.SplitHostPort(c.addr)
	fmt.Printf("Target:%s IP:%s Timestamp:%d Seq:%d\n", c.target, ip, c.timestamp, counter)
	for i := 0; i < v.NumField(); i++ {
		if strings.Contains(filter, strings.ToLower(v.Type().Field(i).Name)) || filter == "" {
			fmt.Printf("%s:%d ", v.Type().Field(i).Name, v.Field(i).Interface())
		}
	}
	fmt.Println("")
}

func (c *client) printJSONPretty(counter int) {
	ip, _, _ := net.SplitHostPort(c.addr)
	b, err := json.MarshalIndent(struct {
		Target    string
		IP        string
		Timestamp int64
		Seq       int
		stats
	}{
		c.target,
		ip,
		c.timestamp,
		counter,
		c.stats,
	}, "", "  ")

	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println(string(b))
}

func (c *client) printJSON(counter int) {
	ip, _, _ := net.SplitHostPort(c.addr)
	b, err := json.Marshal(struct {
		Target    string
		IP        string
		Timestamp int64
		Seq       int
		stats
	}{
		c.target,
		ip,
		c.timestamp,
		counter,
		c.stats,
	})

	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println(string(b))
}
