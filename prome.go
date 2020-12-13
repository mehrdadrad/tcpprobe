package main

import (
	"context"
	"encoding/json"
	"log"
	"reflect"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var reLabel = regexp.MustCompile(`^[a-zA-Z0-9_]*$`)

func (c *client) prometheus(ctx context.Context) {
	var (
		err error
		f   func() float64
	)

	v := reflect.ValueOf(&c.stats).Elem()
	for i := 0; i < v.NumField(); i++ {
		i := i

		if v.Type().Field(i).Tag.Get("unexported") == "true" {
			continue
		}

		switch v.Field(i).Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
			f = func() float64 {
				return float64(v.Field(i).Uint())
			}

		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
			f = func() float64 {
				return float64(v.Field(i).Int())
			}
		case reflect.String:
			continue
		}

		if v.Type().Field(i).Tag.Get("kind") == "counter" {
			err = prometheus.Register(prometheus.NewCounterFunc(prometheus.CounterOpts{
				Name:        "tp_" + v.Type().Field(i).Tag.Get("name"),
				Help:        v.Type().Field(i).Tag.Get("help"),
				ConstLabels: getLabels(ctx, c.target),
			}, f))

		} else {
			err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				Name:        "tp_" + v.Type().Field(i).Tag.Get("name"),
				Help:        v.Type().Field(i).Tag.Get("help"),
				ConstLabels: getLabels(ctx, c.target),
			}, f))
		}

		if err != nil {
			log.Println(err, c.target)
		}
	}

}

func (c *client) deprometheus(ctx context.Context) {
	var (
		ok bool
		f  func() float64
	)

	v := reflect.ValueOf(&c.stats).Elem()
	for i := 0; i < v.NumField(); i++ {
		i := i

		if v.Type().Field(i).Tag.Get("unexported") == "true" {
			continue
		}

		switch v.Field(i).Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
			f = func() float64 {
				return float64(v.Field(i).Uint())
			}

		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
			f = func() float64 {
				return float64(v.Field(i).Int())
			}
		case reflect.String:
			continue
		}

		if v.Type().Field(i).Tag.Get("kind") == "counter" {
			ok = prometheus.Unregister(prometheus.NewCounterFunc(prometheus.CounterOpts{
				Name:        "tp_" + v.Type().Field(i).Tag.Get("name"),
				ConstLabels: getLabels(ctx, c.target),
			}, f))
		} else {
			ok = prometheus.Unregister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				Name:        "tp_" + v.Type().Field(i).Tag.Get("name"),
				ConstLabels: getLabels(ctx, c.target),
			}, f))
		}

		if !ok {
			log.Println("prometheus unregister failed:", c.target)
		}
	}
}

func getLabels(ctx context.Context, target string) prometheus.Labels {
	labels := prometheus.Labels{"target": target}

	if v := ctx.Value(labelsKey); v != nil {
		m := map[string]string{}
		if err := json.Unmarshal(v.([]byte), &m); err != nil {
			return labels
		}
		for k, v := range m {
			k = strings.Replace(k, "-", "_", -1)
			if reLabel.MatchString(k) {
				labels[k] = v
			}
		}
	}

	return labels
}
