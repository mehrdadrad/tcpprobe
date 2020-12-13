package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sethvargo/go-signalcontext"
)

type intervalContextKey string
type labelsContextKey string

type prop struct {
	cancel context.CancelFunc
	client *client
}

type tp struct {
	sync.Mutex
	targets map[string]prop
}

var (
	intervalKey intervalContextKey
	labelsKey   labelsContextKey
)

func main() {
	wg := &sync.WaitGroup{}
	ctx, cancel := signalcontext.OnInterrupt()
	defer cancel()

	req, targets, err := getCli(os.Args)
	if err != nil {
		return
	}

	tp := &tp{targets: make(map[string]prop)}

	wg.Add(len(targets))
	for _, target := range targets {
		go func(target string) {
			defer wg.Done()
			tp.start(ctx, target, req)
			tp.cleanup(ctx, target)
		}(target)
	}

	cfg, err := getConfig(req.config)
	if err != nil {
		log.Fatal(err)
	}

	wg.Add(len(cfg.Targets))
	for _, t := range cfg.Targets {
		go func(ctx context.Context, target target) {
			defer wg.Done()
			b, _ := json.Marshal(target.Labels)
			ctx = context.WithValue(ctx, intervalKey, target.Interval)
			ctx = context.WithValue(ctx, labelsKey, b)
			tp.start(ctx, target.Addr, req)
			tp.cleanup(ctx, target.Addr)
		}(ctx, t)
	}

	if req.k8s {
		kube().start(ctx, tp, req)
	}

	if !req.promDisabled {
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			log.Fatal(http.ListenAndServe(req.promAddr, nil))
		}()
	}

	wait(ctx, wg, req)
}

func wait(ctx context.Context, wg *sync.WaitGroup, req *request) {
	wg.Wait()

	if req.k8s {
		<-ctx.Done()
	}
}

func (t *tp) start(ctx context.Context, target string, req *request) {
	t.Lock()

	if _, ok := t.targets[target]; ok {
		t.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	c := newClient(req, target)
	t.targets[target] = prop{cancel, c}
	t.Unlock()

	c.prometheus(ctx)
	c.probe(ctx)
}

func (t *tp) cleanup(ctx context.Context, target string) {
	t.Lock()
	defer t.Unlock()

	if _, ok := t.targets[target]; !ok {
		return
	}

	t.targets[target].client.deprometheus(ctx)
	delete(t.targets, target)
}

func (t *tp) stop(target string) {
	t.Lock()
	defer t.Unlock()

	if _, ok := t.targets[target]; !ok {
		return
	}

	t.targets[target].cancel()
}
