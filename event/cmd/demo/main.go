package main

import (
	"context"
	"encoding/json"
	"flag"
	"math/rand"
	"os"
	"strconv"
	"time"

	client "github.com/dizaknz/gkgraph/event/client/grpc"
	"github.com/dizaknz/gkgraph/event/pkg/types"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/transport/grpc"
	gogrpc "google.golang.org/grpc"
)

func main() {
	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.DefaultCaller)

	fs := flag.NewFlagSet("demo", flag.ExitOnError)
	url := fs.String("service", "localhost:8082", "URL for gRPC event services")
	msg := fs.Int("messages", 100, "Number of demo events to send to gRPC event server")

	fs.Parse(os.Args[1:])

	conn, err := gogrpc.Dial(
		*url,
		gogrpc.WithInsecure(),
		gogrpc.WithTimeout(time.Second),
	)
	if err != nil {
		level.Error(logger).Log(
			"msg", "failed to connect to server",
			"error", err,
		)
		os.Exit(1)
	}
	defer conn.Close()

	svc, err := client.New(conn, map[string][]grpc.ClientOption{})
	if err != nil {
		level.Error(logger).Log(
			"msg", "failed to create client",
			"error", err,
		)
		os.Exit(1)
	}
	eventTypes := []string{
		"TYPE1",
		"TYPE2",
		"TYPE3",
		"TYPE4",
	}
	linkTypes := []string{
		"CHILD_OF",
		"RELATED_TO",
	}
	events := map[int]string{}
	for i := 0; i < *msg; i++ {
		ev := types.Event{
			ID:        strconv.Itoa(i),
			Type:      eventTypes[rand.Intn(len(eventTypes))],
			Timestamp: time.Now().UTC(),
			Attributes: []*types.Attribute{
				{
					Name:  "name",
					Value: RandStringBytes(10),
				},
				{
					Name:  "sid",
					Value: RandStringBytes(20),
				},
			},
		}
		// create random links for 75% of nodes
		if i > (*msg / 4) {
			eventID := rand.Intn(i)
			ev.Links = []*types.EventLink{
				{
					EventID:   strconv.Itoa(eventID),
					EventType: events[eventID],
					LinkType:  types.NewLinkType(linkTypes[rand.Intn(len(linkTypes))]),
					Attributes: []*types.Attribute{
						{
							Name:  "desc",
							Value: RandStringBytes(10),
						},
					},
				},
			}
			eventID = rand.Intn(i)
			ev.Links = append(
				ev.Links,
				&types.EventLink{
					EventID:   strconv.Itoa(eventID),
					EventType: events[eventID],
					LinkType:  types.NewLinkType(linkTypes[rand.Intn(len(linkTypes))]),
				},
			)
		}
		events[i] = ev.Type
		b, _ := json.Marshal(ev)
		level.Debug(logger).Log("Sending", string(b))
		err := svc.Add(context.Background(), &ev)
		if err != nil {
			level.Error(logger).Log(
				"msg", "failed to create event",
				"error", err,
			)
			continue
		}
	}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
