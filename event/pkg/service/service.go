package service

import (
	"context"
	"errors"

	"github.com/dizaknz/gkgraph/event/pkg/datastore"
	"github.com/dizaknz/gkgraph/event/pkg/types"
)

type EventService interface {
	Add(ctx context.Context, event *types.Event) error
}

type eventService struct {
	ds *datastore.EventDatastore
}

func (e *eventService) Add(ctx context.Context, event *types.Event) error {
	if event == nil {
		return errors.New("no event provided")
	}
	return e.ds.Add(event)
}

func NewEventService(ds *datastore.EventDatastore) EventService {
	return &eventService{
		ds: ds,
	}
}

func New(ds *datastore.EventDatastore, middleware []Middleware) EventService {
	var svc EventService = NewEventService(ds)
	for _, m := range middleware {
		svc = m(svc)
	}
	return svc
}
