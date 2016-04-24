package main

import (
	"github.com/lysu/go-misc/grpc/hier/protos"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"net"
)

const (
	port = ":50051"
)

type server struct{}

func (s *server) SayHi(ctx context.Context, req *protos.Req) (*protos.Resp, error) {
	return &protos.Resp{Message: "hey~~~~" + req.Name}, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	protos.RegisterHierServer(s, &server{})
	s.Serve(lis)
}
