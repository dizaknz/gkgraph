package grpc

import (
	"context"
	"errors"
	"fmt"

	endpoint1 "github.com/dizaknz/gkgraph/event/pkg/endpoint"
	"github.com/dizaknz/gkgraph/event/pkg/grpc/pb"
	"github.com/dizaknz/gkgraph/event/pkg/service"

	"github.com/go-kit/kit/endpoint"
	grpc1 "github.com/go-kit/kit/transport/grpc"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
)

func New(conn *grpc.ClientConn, options map[string][]grpc1.ClientOption) (service.EventService, error) {
	var addEndpoint endpoint.Endpoint
	{
		addEndpoint = grpc1.NewClient(
			conn,
			"pb.Event",
			"Add",
			encodeAddRequest,
			decodeAddResponse,
			pb.AddReply{},
			options["Add"]...,
		).Endpoint()
	}

	return endpoint1.Endpoints{AddEndpoint: addEndpoint}, nil
}

func encodeAddRequest(_ context.Context, r interface{}) (interface{}, error) {
	in := r.(endpoint1.AddRequest)
	ts, err := ptypes.TimestampProto(in.Event.Timestamp)
	if err != nil {
		return nil, err
	}
	msg := &pb.EventMessage{
		Id:        in.Event.ID,
		Typ:       in.Event.Type,
		Timestamp: ts,
	}
	if len(in.Event.Attributes) > 0 {
		attrs := make([]*pb.AttrValue, len(in.Event.Attributes))
		for i, av := range in.Event.Attributes {
			attrs[i] = &pb.AttrValue{
				Attr: av.Name,
				Val:  fmt.Sprintf("%s", av.Value),
			}
		}
		msg.Attrs = attrs
	}
	if len(in.Event.Links) > 0 {
		links := make([]*pb.EventLink, len(in.Event.Links))
		for i, link := range in.Event.Links {
			links[i] = &pb.EventLink{
				EventID:   link.EventID,
				EventType: link.EventType,
				LinkType:  link.LinkType.String(),
			}
			if len(link.Attributes) > 0 {
				attrs := make([]*pb.AttrValue, len(link.Attributes))
				for i, av := range link.Attributes {
					attrs[i] = &pb.AttrValue{
						Attr: av.Name,
						Val:  fmt.Sprintf("%s", av.Value),
					}
				}
				links[i].Attrs = attrs
			}
		}
		msg.Links = links
	}

	return &pb.AddRequest{
		Event: msg,
	}, nil
}

func decodeAddResponse(_ context.Context, r interface{}) (interface{}, error) {
	out := r.(*pb.AddReply)
	if !out.Success {
		return endpoint1.AddResponse{
			Error: errors.New(out.Message),
		}, nil
	}
	return endpoint1.AddResponse{
		Error: nil,
	}, nil
}
