package endpoint

import (
	"context"

	"github.com/dizaknz/gkgraph/event/pkg/service"
	"github.com/dizaknz/gkgraph/event/pkg/types"

	"github.com/go-kit/kit/endpoint"
)

type AddRequest struct {
	Event *types.Event `json:"event"`
}

type AddResponse struct {
	Error error `json:"error"`
}

func MakeAddEndpoint(s service.EventService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AddRequest)
		e0 := s.Add(ctx, req.Event)
		return AddResponse{Error: e0}, nil
	}
}

func (r AddResponse) Failed() error {
	return r.Error
}

type Failure interface {
	Failed() error
}

func (e Endpoints) Add(ctx context.Context, event *types.Event) error {
	request := AddRequest{Event: event}
	response, err := e.AddEndpoint(ctx, request)
	if err != nil {
		return err
	}
	return response.(AddResponse).Error
}
