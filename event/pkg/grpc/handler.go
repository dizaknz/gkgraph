package grpc

import (
	"context"

	"github.com/dizaknz/gkgraph/event/pkg/endpoint"
	"github.com/dizaknz/gkgraph/event/pkg/grpc/pb"
	"github.com/dizaknz/gkgraph/event/pkg/types"

	"github.com/go-kit/kit/transport/grpc"
	"github.com/golang/protobuf/ptypes"
)

func makeAddHandler(endpoints endpoint.Endpoints, options []grpc.ServerOption) grpc.Handler {
	return grpc.NewServer(
		endpoints.AddEndpoint,
		decodeAddRequest,
		encodeAddResponse,
		options...,
	)
}

func decodeAddRequest(_ context.Context, r interface{}) (interface{}, error) {
	in := r.(*pb.AddRequest)

	ts, err := ptypes.Timestamp(in.Event.Timestamp)
	if err != nil {
		return nil, err
	}
	ev := &types.Event{
		ID:        in.Event.Id,
		Type:      in.Event.Typ,
		Timestamp: ts,
	}
	attrs := []*types.Attribute{}
	for _, av := range in.Event.Attrs {
		attrs = append(attrs, &types.Attribute{
			Name:  av.Attr,
			Value: av.Val,
		})
	}
	ev.Attributes = attrs
	if len(in.Event.Links) > 0 {
		links := make([]*types.EventLink, len(in.Event.Links))
		for i, link := range in.Event.Links {
			links[i] = &types.EventLink{
				EventID:   link.EventID,
				EventType: link.EventType,
				LinkType:  types.NewLinkType(link.LinkType),
			}
			if len(link.Attrs) > 0 {
				links[i].Attributes = make([]*types.Attribute, len(link.Attrs))
				for j, av := range link.Attrs {
					links[i].Attributes[j] = &types.Attribute{
						Name:  av.Attr,
						Value: av.Val,
					}
				}
			}
		}
		ev.Links = links
	}
	return endpoint.AddRequest{
		Event: ev,
	}, nil
}

func encodeAddResponse(_ context.Context, r interface{}) (interface{}, error) {
	out := r.(endpoint.AddResponse)
	if out.Error != nil {
		return &pb.AddReply{
			Success: false,
			Message: out.Error.Error(),
		}, nil
	}
	return &pb.AddReply{
		Success: true,
		Message: "Successfully added event",
	}, nil
}

func (g *grpcServer) Add(ctx context.Context, req *pb.AddRequest) (*pb.AddReply, error) {
	_, rep, err := g.add.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.AddReply), nil
}
