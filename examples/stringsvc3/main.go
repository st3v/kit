package main

import (
	"flag"
	"net/http"
	"os"
	"strings"
	"time"

	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/loadbalancer"
	"github.com/go-kit/kit/loadbalancer/static"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	httptransport "github.com/go-kit/kit/transport/http"
)

func main() {
	var (
		listen = flag.String("listen", ":8080", "HTTP listen address")
		proxy  = flag.String("proxy", "", "Optional URL to proxy uppercase requests")
	)
	flag.Parse()

	ctx := context.Background()
	logger := log.NewLogfmtLogger(os.Stderr)

	fieldKeys := []string{"method", "error"}
	requestCount := kitprometheus.NewCounter(stdprometheus.CounterOpts{
		Namespace: "my_group",
		Subsystem: "string_service",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)
	requestLatency := metrics.NewTimeHistogram(time.Microsecond, kitprometheus.NewSummary(stdprometheus.SummaryOpts{
		Namespace: "my_group",
		Subsystem: "string_service",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of requests in microseconds.",
	}, fieldKeys))
	countResult := kitprometheus.NewSummary(stdprometheus.SummaryOpts{
		Namespace: "my_group",
		Subsystem: "string_service",
		Name:      "count_result",
		Help:      "The result of each count method.",
	}, []string{}) // no fields here

	var svc StringService
	svc = stringService{}
	svc = loggingMiddleware{logger, svc}
	svc = instrumentingMiddleware{requestCount, requestLatency, countResult, svc}

	var uppercase endpoint.Endpoint
	if *proxy != "" {
		var (
			publisher   = static.NewPublisher(split(*proxy), factory, logger) // could use any Publisher here
			lb          = loadbalancer.NewRoundRobin(publisher)
			maxAttempts = 3
			maxTime     = 100 * time.Millisecond
		)
		uppercase = loadbalancer.Retry(maxAttempts, maxTime, lb)
	} else {
		uppercase = makeUppercaseEndpoint(svc)
	}

	uppercaseHandler := httptransport.Server{
		Context:    ctx,
		Endpoint:   uppercase,
		DecodeFunc: decodeUppercaseRequest,
		EncodeFunc: encodeResponse,
	}

	countHandler := httptransport.Server{
		Context:    ctx,
		Endpoint:   makeCountEndpoint(svc),
		DecodeFunc: decodeCountRequest,
		EncodeFunc: encodeResponse,
	}

	http.Handle("/uppercase", uppercaseHandler)
	http.Handle("/count", countHandler)
	_ = logger.Log("msg", "HTTP", "addr", *listen)
	_ = logger.Log("err", http.ListenAndServe(*listen, nil))
}

func split(s string) []string {
	a := strings.Split(s, ",")
	for i := range a {
		a[i] = strings.TrimSpace(a[i])
	}
	return a
}
