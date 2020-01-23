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
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	grpc1 "google.golang.org/grpc"

	grpc2 "github.com/go-kit/kit/transport/grpc"
	http2 "github.com/go-kit/kit/transport/http"
)

func main() {
	viper.SetEnvPrefix("gkevent")
	fs := flag.NewFlagSet("gkevent", flag.ExitOnError)
	fs.String("debug_url", ":8080", "Debug and metrics URL")
	fs.String("http_url", ":8081", "HTTP URL")
	fs.String("grpc_url", ":8082", "gRPC URL")
	fs.String("neo4j", "bolt://username:password@localhost:7687", "Neo4j bolt")
	for _, env := range []string{
		"debug_url",
		"http_url",
		"grpc_url",
		"neo4j",
	} {
		viper.BindEnv(env)
	}

	pflag.CommandLine.AddGoFlagSet(fs)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	debugURL := viper.GetString("debug_url")
	httpURL := viper.GetString("http_url")
	grpcURL := viper.GetString("grpc_url")
	neo4j := viper.GetString("neo4j")

	banner()

	logger := log.NewLogfmtLogger(os.Stderr)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.DefaultCaller)

	tracer := opentracinggo.GlobalTracer()

	level.Debug(logger).Log("msg", "Connecting to Neo4j")
	db, err := connect(neo4j)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to connect to Neo4j", "error", err)
		return
	}
	defer db.Close()

	svc := service.New(
		datastore.NewEventDatastore(db, logger),
		getServiceMiddleware(logger),
	)
	eps := endpoint.New(svc, getEndpointMiddleware(logger))
	g := createService(httpURL, grpcURL, eps, logger, tracer)
	initMetricsEndpoint(debugURL, g, logger)
	initCancelInterrupt(g)
	logger.Log("exit", g.Run())
}

func connect(uri string) (bolt.Conn, error) {
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
	httpURL, grpcURL string,
	endpoints endpoint.Endpoints,
	logger log.Logger,
	tracer opentracinggo.Tracer,
) (g *group.Group) {
	g = &group.Group{}
	initHttpHandler(httpURL, endpoints, g, logger, tracer)
	initGRPCHandler(grpcURL, endpoints, g, logger, tracer)
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

func banner() {
	fmt.Printf(`
      ___           ___           ___           ___           ___           ___           ___           ___     
     /\  \         /\__\         /\  \         /\__\         /\  \         /\__\         /\  \         /\  \    
    /::\  \       /:/  /        /::\  \       /:/  /        /::\  \       /::|  |        \:\  \       /::\  \   
   /:/\:\  \     /:/__/        /:/\:\  \     /:/  /        /:/\:\  \     /:|:|  |         \:\  \     /:/\ \  \  
  /:/  \:\  \   /::\__\____   /::\~\:\  \   /:/__/  ___   /::\~\:\  \   /:/|:|  |__       /::\  \   _\:\~\ \  \ 
 /:/__/_\:\__\ /:/\:::::\__\ /:/\:\ \:\__\  |:|  | /\__\ /:/\:\ \:\__\ /:/ |:| /\__\     /:/\:\__\ /\ \:\ \ \__\
 \:\  /\ \/__/ \/_|:|~~|~    \:\~\:\ \/__/  |:|  |/:/  / \:\~\:\ \/__/ \/__|:|/:/  /    /:/  \/__/ \:\ \:\ \/__/
  \:\ \:\__\      |:|  |      \:\ \:\__\    |:|__/:/  /   \:\ \:\__\       |:/:/  /    /:/  /       \:\ \:\__\  
   \:\/:/  /      |:|  |       \:\ \/__/     \::::/__/     \:\ \/__/       |::/  /     \/__/         \:\/:/  /  
    \::/  /       |:|  |        \:\__\        ~~~~          \:\__\         /:/  /                     \::/  /   
     \/__/         \|__|         \/__/                       \/__/         \/__/                       \/__/    

	`)
}
