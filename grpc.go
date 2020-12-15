package main

import (
	"context"
	"encoding/json"
	"log"
	"net"

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
		resp *pb.Response
		err  error
	)

	conn, err := grpc.Dial(req.cmd.addr, grpc.WithInsecure())
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
			resp, err = c.Add(context.TODO(), pt)

		} else {
			resp, err = c.Delete(context.TODO(), pt)

		}

		if err != nil {
			log.Println(err)
		} else {
			log.Printf("message: %s, code: %d", resp.Message, resp.Code)
		}
	}
}
