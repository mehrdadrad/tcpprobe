package main

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type k8s struct {
	clientset kubernetes.Interface
	pods      sync.Map
}

func kube() *k8s {
	cs, err := newClientset()
	if err != nil {
		log.Fatal(err)
	}

	k := &k8s{
		clientset: cs,
		pods:      sync.Map{},
	}
	return k
}

func (k *k8s) start(ctx context.Context, tp *tp, req *request) {
	go func() {
		for {
			pods, err := k.clientset.CoreV1().Pods(req.namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Println(err)
				time.Sleep(time.Second)
				continue
			}

			for _, pod := range pods.Items {
				if _, ok := k.pods.Load(pod.Name); !ok && pod.Status.Phase == "Running" {
					k.pods.Store(pod.Name, pod.Status.PodIP)
					for _, target := range getTargets(&pod) {
						if ok := tp.isExist(target); ok {
							log.Println(errExist, target)
							continue
						}
						go func(ctx context.Context, pod v1.Pod, target string) {
							ctx = context.WithValue(ctx, intervalKey, pod.Annotations["tcpprobe/interval"])
							ctx = context.WithValue(ctx, labelsKey, []byte(pod.Annotations["tcpprobe/labels"]))
							tp.start(ctx, target, req)
							tp.cleanup(ctx, target)
						}(ctx, pod, target)

						log.Printf("pod: %s, target: %s has been added", pod.Name, target)
					}
				}
			}
			<-time.After(5 * time.Second)
		}
	}()

	factory := informers.NewSharedInformerFactoryWithOptions(k.clientset, time.Second*5, informers.WithNamespace(req.namespace))
	informer := factory.Core().V1().Pods().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			pod, ok := obj.(*v1.Pod)
			if !ok {
				return
			}

			podIP, ok := k.pods.Load(pod.Name)
			if !ok {
				return
			}

			pod.Status.PodIP = podIP.(string)
			for _, target := range getTargets(pod) {
				log.Printf("pod: %s, target: %s has been deleted", pod.Name, target)
				tp.stop(target)
			}
			k.pods.Delete(pod.Name)
		},
	})

	stop := make(chan struct{})
	go informer.Run(stop)
	log.Println("k8s has been started")
}

func newClientset() (*kubernetes.Clientset, error) {
	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(clusterConfig)
}

func getTargets(n *v1.Pod) []string {
	targets, ok := n.Annotations["tcpprobe/targets"]
	if !ok {
		return []string{}
	}

	targets = strings.Replace(targets, "PODIP", n.Status.PodIP, -1)
	return strings.Split(targets, ";;")
}
