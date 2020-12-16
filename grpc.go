package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"reflect"
	"time"

	pbstruct "github.com/golang/protobuf/ptypes/struct"
	pb "github.com/mehrdadrad/tcpprobe/proto"
	"google.golang.org/grpc"
)

type gServer struct {
	tp  *tp
	req *request
}

func (g *gServer) Add(ctx context.Context, target *pb.Target) (*pb.Response, error) {
	if ok := g.tp.isExist(target.Addr); ok {
		return &pb.Response{Message: errExist.Error(), Code: 400}, nil
	}

	go func() {
		ctx := context.Background()
		b, _ := json.Marshal(target.Labels)
		ctx = context.WithValue(ctx, intervalKey, target.Interval)
		ctx = context.WithValue(ctx, labelsKey, b)
		g.tp.start(ctx, target.Addr, g.req)
		g.tp.cleanup(ctx, target.Addr)
	}()

	return &pb.Response{Message: "target has been added", Code: 200}, nil
}

func (g *gServer) Delete(ctx context.Context, target *pb.Target) (*pb.Response, error) {
	if ok := g.tp.isExist(target.Addr); !ok {
		return &pb.Response{Message: "target is not exist", Code: 404}, nil
	}

	g.tp.stop(target.Addr)

	return &pb.Response{Message: "target has been deleted", Code: 200}, nil
}

func (g *gServer) Get(target *pb.Target, stream pb.TCPProbe_GetServer) error {
	var (
		t  prop
		ok bool
	)

	if t, ok = g.tp.targets[target.GetAddr()]; !ok {
		return fmt.Errorf("target: %s not exist", target.GetAddr())
	}

	ch := make(chan *stats, 1)

	t.client.subscribe(ch)
	defer t.client.unsubscribe(ch)

	for {
		stats, ok := <-ch
		if !ok {
			break
		}

		err := stream.Send(
			&pb.Stats{
				Metrics: stats2pbStruct(stats),
			},
		)
		if err != nil {
			break
		}
	}

	return nil
}

func grpcServer(tp *tp, req *request) {
	l, err := net.Listen("tcp", req.grpcAddr)
	if err != nil {
		log.Fatal(err)
	}

	srv := &gServer{tp: tp, req: req}
	s := grpc.NewServer()
	pb.RegisterTCPProbeServer(s, srv)
	go func() {
		log.Fatal(s.Serve(l))
	}()
}

func grpcClient(req *request) {
	var (
		opts = []grpc.DialOption{}
		resp *pb.Response
		err  error
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if req.cmd.insecure {
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(req.cmd.addr, opts...)
	if err != nil {
		log.Fatal(err)
	}

	c := pb.NewTCPProbeClient(conn)

	for _, target := range req.cmd.args {
		labels := map[string]string{}
		json.Unmarshal([]byte(req.cmd.labels), &labels)

		pt := &pb.Target{
			Addr:     target,
			Interval: req.cmd.interval,
			Labels:   labels,
		}
		if req.cmd.cmd != "del" {
			resp, err = c.Add(ctx, pt)

		} else {
			resp, err = c.Delete(ctx, pt)
		}

		if err != nil {
			log.Println(err)
		} else {
			log.Printf("message: %s, code: %d", resp.Message, resp.Code)
		}
	}
}

func stats2pbStruct(stats *stats) *pbstruct.Struct {
	r := &pbstruct.Struct{Fields: make(map[string]*pbstruct.Value)}

	s := reflect.ValueOf(stats).Elem()
	for i := 0; i < s.NumField(); i++ {
		unexported := s.Type().Field(i).Tag.Get("unexported")
		if unexported == "true" {
			continue
		}

		switch s.Type().Field(i).Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			r.Fields[s.Type().Field(i).Name] = &pbstruct.Value{
				Kind: &pbstruct.Value_NumberValue{NumberValue: float64(s.Field(i).Int())},
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			r.Fields[s.Type().Field(i).Name] = &pbstruct.Value{
				Kind: &pbstruct.Value_NumberValue{NumberValue: float64(s.Field(i).Uint())},
			}
		case reflect.String:
			r.Fields[s.Type().Field(i).Name] = &pbstruct.Value{
				Kind: &pbstruct.Value_StringValue{StringValue: s.Field(i).String()},
			}
		}

	}

	return r
}
