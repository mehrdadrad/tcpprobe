package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sethvargo/go-signalcontext"
)

var (
	version      = "0.2.5"
	tpReleaseURL = "https://github.com/mehrdadrad/tcpprobe/releases/latest"
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

	errExist = errors.New("the target already exist")
)

func main() {
	wg := &sync.WaitGroup{}
	ctx, cancel := signalcontext.OnInterrupt()
	defer cancel()

	req, targets, err := getCli(os.Args)
	if err != nil {
		return
	}

	if req.cmd != nil {
		grpcClient(req)
		return
	}

	tp := &tp{targets: make(map[string]prop)}

	// command line targets
	wg.Add(len(targets))
	for _, target := range targets {
		if ok := tp.isExist(target); ok {
			log.Println(errExist, target)
			continue
		}

		go func(target string) {
			defer wg.Done()
			tp.start(ctx, target, req)
			tp.cleanup(ctx, target)
		}(target)
	}

	// config
	cfg, err := getConfig(req.config)
	if err != nil {
		log.Fatal(err)
	}

	wg.Add(len(cfg.Targets))
	for _, t := range cfg.Targets {
		if ok := tp.isExist(t.Addr); ok {
			log.Println(errExist, t.Addr)
			continue
		}

		go func(ctx context.Context, target target) {
			defer wg.Done()
			b, _ := json.Marshal(target.Labels)
			ctx = context.WithValue(ctx, intervalKey, target.Interval)
			ctx = context.WithValue(ctx, labelsKey, b)
			tp.start(ctx, target.Addr, req)
			tp.cleanup(ctx, target.Addr)
		}(ctx, t)
	}

	// kubernetes
	if req.k8s {
		kube().start(ctx, tp, req)
	}

	// grpc server
	if req.grpc {
		grpcServer(tp, req)
	}

	// prometheus
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

	if req.k8s || req.grpc {
		<-ctx.Done()
	}
}

func (t *tp) start(ctx context.Context, target string, req *request) {
	t.Lock()

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

	for _, ch := range t.targets[target].client.subCh {
		close(ch)
	}

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

func (t *tp) isExist(target string) bool {
	t.Lock()
	defer t.Unlock()

	_, ok := t.targets[target]

	return ok
}

func checkUpdate(tpReleaseURL string) (bool, string) {
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("redirect")
		},
	}

	resp, err := client.Get(tpReleaseURL)
	if err != nil && resp != nil && resp.StatusCode == http.StatusFound {
		url, err := resp.Location()
		if err != nil {
			return false, ""
		}
		path := strings.Split(url.Path, "/")
		if path[len(path)-1] == "v"+version {
			return false, ""
		}
		return true, path[len(path)-1]
	}

	return false, ""
}
