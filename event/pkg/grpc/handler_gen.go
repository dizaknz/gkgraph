// THIS FILE IS AUTO GENERATED BY GK-CLI DO NOT EDIT!!
package grpc

import (
	endpoint "github.com/dizaknz/gkgraph/event/pkg/endpoint"
	pb "github.com/dizaknz/gkgraph/event/pkg/grpc/pb"
	grpc "github.com/go-kit/kit/transport/grpc"
)

// NewGRPCServer makes a set of endpoints available as a gRPC AddServer
type grpcServer struct {
	add grpc.Handler
}

func NewGRPCServer(endpoints endpoint.Endpoints, options map[string][]grpc.ServerOption) pb.EventServer {
	return &grpcServer{add: makeAddHandler(endpoints, options["Add"])}
}
