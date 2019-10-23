package service

import (
	"context"

	"github.com/go-kit/kit/log"

	"github.com/dizaknz/gkgraph/event/pkg/types"
)

type Middleware func(EventService) EventService

type loggingMiddleware struct {
	logger log.Logger
	next   EventService
}

func LoggingMiddleware(logger log.Logger) Middleware {
	return func(next EventService) EventService {
		return &loggingMiddleware{logger, next}
	}

}

func (l loggingMiddleware) Add(ctx context.Context, event *types.Event) error {
	defer func() {
		if event != nil {
			l.logger.Log("method", "Add", "event", event.ID)
		}
	}()
	return l.next.Add(ctx, event)
}
