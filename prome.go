package main

import (
	"net/http"
	"reflect"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (c *client) prometheus() {
	v := reflect.ValueOf(&c.stats).Elem()
	for i := 0; i < v.NumField(); i++ {
		i := i

		switch v.Field(i).Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
			prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				Name: v.Type().Field(i).Name,
			},
				func() float64 {
					return float64(v.Field(i).Uint())
				}))
		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
			prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				Name: v.Type().Field(i).Name,
			},
				func() float64 {
					return float64(v.Field(i).Int())
				}))
		}
	}

	http.Handle("/", promhttp.Handler())
	go http.ListenAndServe(c.req.promAddr, nil)
}
