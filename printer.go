package main

import (
	"fmt"
	"reflect"
)

func (c *client) printer() {
	c.printText()
}

func (c *client) printText() {
	v := reflect.ValueOf(c.stats)
	for i := 0; i < v.NumField(); i++ {
		fmt.Printf("%s:%d ", v.Type().Field(i).Name, v.Field(i).Interface())
	}
	fmt.Println("")
}

func (c *client) json() {

}
