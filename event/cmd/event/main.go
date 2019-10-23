package main

import (
	"flag"
	"fmt"
	"net"
	http1 "net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/dizaknz/gkgraph/event/pkg/datastore"
	"github.com/dizaknz/gkgraph/event/pkg/endpoint"
	"github.com/dizaknz/gkgraph/event/pkg/grpc"
	"github.com/dizaknz/gkgraph/event/pkg/grpc/pb"
	"github.com/dizaknz/gkgraph/event/pkg/http"
	"github.com/dizaknz/gkgraph/event/pkg/service"

	endpoint1 "github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-kit/kit/tracing/opentracing"
	bolt "github.com/johnnadratowski/golang-neo4j-bolt-driver"
	"github.com/oklog/oklog/pkg/group"
	opentracinggo "github.com/opentracing/opentracing-go"
	prometheus1 "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	grpc1 "google.golang.org/grpc"

	grpc2 "github.com/go-kit/kit/transport/grpc"
	http2 "github.com/go-kit/kit/transport/http"
)

func main() {
	var tracer opentracinggo.Tracer
	var logger log.Logger

	var fs = flag.NewFlagSet("event", flag.ExitOnError)
	var debugAddr = fs.String("debug-addr", ":8080", "Debug and metrics listen address")
	var httpAddr = fs.String("http-addr", ":8081", "HTTP listen address")
	var grpcAddr = fs.String("grpc-addr", ":8082", "gRPC listen address")
	var neo4j = fs.String("neo4j", "bolt://username:password@localhost:7687", "Neo4j bolt")

	fs.Parse(os.Args[1:])

	logger = log.NewLogfmtLogger(os.Stderr)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.DefaultCaller)

	tracer = opentracinggo.GlobalTracer()

	db, err := createConnection(*neo4j)
	if err != nil {
		level.Error(logger).Log("Neo4j", "Failed to connect", "error", err)
		return
	}
	defer db.Close()

	svc := service.New(
		datastore.NewEventDatastore(db, logger),
		getServiceMiddleware(logger),
	)
	eps := endpoint.New(svc, getEndpointMiddleware(logger))
	g := createService(*httpAddr, *grpcAddr, eps, logger, tracer)
	initMetricsEndpoint(*debugAddr, g, logger)
	initCancelInterrupt(g)
	logger.Log("exit", g.Run())

}

func createConnection(uri string) (bolt.Conn, error) {
	return bolt.NewDriver().OpenNeo(uri)
}

func initHttpHandler(
	addr string,
	endpoints endpoint.Endpoints,
	g *group.Group,
	logger log.Logger,
	tracer opentracinggo.Tracer,
) {
	options := defaultHttpOptions(logger, tracer)

	httpHandler := http.NewHTTPHandler(endpoints, options)
	httpListener, err := net.Listen("tcp", addr)
	if err != nil {
		level.Error(logger).Log("transport", "HTTP", "during", "Listen", "error", err)
	}
	g.Add(func() error {
		level.Debug(logger).Log("transport", "HTTP", "addr", addr)
		return http1.Serve(httpListener, httpHandler)
	}, func(error) {
		httpListener.Close()
	})

}
func getServiceMiddleware(logger log.Logger) (mw []service.Middleware) {
	mw = []service.Middleware{}
	mw = addDefaultServiceMiddleware(logger, mw)

	return
}

func getEndpointMiddleware(logger log.Logger) (mw map[string][]endpoint1.Middleware) {
	mw = map[string][]endpoint1.Middleware{}
	duration := prometheus.NewSummaryFrom(prometheus1.SummaryOpts{
		Help:      "Request duration in seconds.",
		Name:      "request_duration_seconds",
		Namespace: "gkgraph",
		Subsystem: "event",
	}, []string{"method", "success"})
	addDefaultEndpointMiddleware(logger, duration, mw)

	return
}

func initMetricsEndpoint(addr string, g *group.Group, logger log.Logger) {
	http1.DefaultServeMux.Handle("/metrics", promhttp.Handler())
	debugListener, err := net.Listen("tcp", addr)
	if err != nil {
		level.Debug(logger).Log("transport", "debug/HTTP", "during", "Listen", "error", err)
	}
	g.Add(func() error {
		logger.Log("transport", "debug/HTTP", "addr", addr)
		return http1.Serve(debugListener, http1.DefaultServeMux)
	}, func(error) {
		debugListener.Close()
	})
}

func initCancelInterrupt(g *group.Group) {
	cancelInterrupt := make(chan struct{})
	g.Add(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		select {
		case sig := <-c:
			return fmt.Errorf("received signal %s", sig)
		case <-cancelInterrupt:
			return nil
		}
	}, func(error) {
		close(cancelInterrupt)
	})
}

func initGRPCHandler(
	addr string,
	endpoints endpoint.Endpoints,
	g *group.Group,
	logger log.Logger,
	tracer opentracinggo.Tracer,
) {
	options := defaultGRPCOptions(logger, tracer)

	grpcServer := grpc.NewGRPCServer(endpoints, options)
	grpcListener, err := net.Listen("tcp", addr)
	if err != nil {
		level.Error(logger).Log("transport", "gRPC", "during", "Listen", "error", err)
	}
	g.Add(func() error {
		level.Debug(logger).Log("transport", "gRPC", "addr", addr)
		baseServer := grpc1.NewServer()
		pb.RegisterEventServer(baseServer, grpcServer)
		return baseServer.Serve(grpcListener)
	}, func(error) {
		grpcListener.Close()
	})

}

func createService(
	httpAddr, grpcAddr string,
	endpoints endpoint.Endpoints,
	logger log.Logger,
	tracer opentracinggo.Tracer,
) (g *group.Group) {
	g = &group.Group{}
	initHttpHandler(httpAddr, endpoints, g, logger, tracer)
	initGRPCHandler(grpcAddr, endpoints, g, logger, tracer)
	return g
}

func defaultHttpOptions(
	logger log.Logger,
	tracer opentracinggo.Tracer,
) map[string][]http2.ServerOption {
	options := map[string][]http2.ServerOption{
		"Add": {
			http2.ServerErrorEncoder(http.ErrorEncoder),
			http2.ServerErrorLogger(logger),
			http2.ServerBefore(opentracing.HTTPToContext(tracer, "Add", logger)),
		},
	}
	return options
}

func defaultGRPCOptions(
	logger log.Logger,
	tracer opentracinggo.Tracer,
) map[string][]grpc2.ServerOption {
	options := map[string][]grpc2.ServerOption{
		"Add": {
			grpc2.ServerErrorLogger(logger),
			grpc2.ServerBefore(opentracing.GRPCToContext(tracer, "Add", logger)),
		},
	}
	return options
}

func addDefaultEndpointMiddleware(
	logger log.Logger,
	duration *prometheus.Summary,
	mw map[string][]endpoint1.Middleware,
) {
	mw["Add"] = []endpoint1.Middleware{
		endpoint.LoggingMiddleware(log.With(logger, "method", "Add")),
		endpoint.InstrumentingMiddleware(duration.With("method", "Add")),
	}
}

func addDefaultServiceMiddleware(logger log.Logger, mw []service.Middleware) []service.Middleware {
	return append(mw, service.LoggingMiddleware(logger))
}
