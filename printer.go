package main

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
)

func (c *client) printer() {
	if !c.req.json {
		c.printText()
	} else {
		c.printJSON()
	}
}

func (c *client) printText() {
	v := reflect.ValueOf(c.stats)

	fmt.Printf("Target:%s IP:%s Timestamp:%d\n", c.target, c.addr, c.timestamp)
	for i := 0; i < v.NumField(); i++ {
		fmt.Printf("%s:%d ", v.Type().Field(i).Name, v.Field(i).Interface())
	}
	fmt.Println("")
}

func (c *client) printJSON() {
	b, err := json.MarshalIndent(struct {
		Target    string
		IP        string
		Timestamp int64
		stats
	}{
		c.target,
		c.addr,
		c.timestamp,
		c.stats,
	}, "", "  ")

	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println(string(b))
}
