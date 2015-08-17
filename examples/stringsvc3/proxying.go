package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	jujuratelimit "github.com/juju/ratelimit"
	"github.com/sony/gobreaker"
	"golang.org/x/net/context"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/loadbalancer"
	"github.com/go-kit/kit/loadbalancer/static"
	"github.com/go-kit/kit/log"
	kitratelimit "github.com/go-kit/kit/ratelimit"
)

func proxyingMiddleware(proxyList string, ctx context.Context, logger log.Logger) func(StringService) StringService {
	return func(next StringService) StringService {
		var (
			publisher   = static.NewPublisher(split(proxyList), factory, logger) // could use any Publisher here
			lb          = loadbalancer.NewRoundRobin(publisher)
			maxAttempts = 3
			maxTime     = 100 * time.Millisecond
			endpoint    = loadbalancer.Retry(maxAttempts, maxTime, lb)
		)
		return proxymw{ctx, endpoint, next}
	}
}

// proxymw implements StringService, forwarding Uppercase requests to the
// provided endpoint, and serving all other (i.e. Count) requests via the
// embedded StringService.
type proxymw struct {
	context.Context
	UppercaseEndpoint endpoint.Endpoint
	StringService
}

func (mw proxymw) Uppercase(s string) (string, error) {
	// Translate business-domain to endpoint-domain.
	response, err := mw.UppercaseEndpoint(mw.Context, uppercaseRequest{S: s})
	if err != nil {
		return "", err
	}

	// Translate endpoint-domain to business-domain.
	resp := response.(uppercaseResponse)
	return resp.V, resp.Err
}

// factory is an endpoint factory for Uppercase RPCs.
func factory(instance string) (endpoint.Endpoint, error) {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}

	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	if u.Path == "" {
		u.Path = "/uppercase"
	}

	// Each individual instance should be wrapped with our circuit breaker and
	// rate limiter. Otherwise, we don't really reap any benefit.
	var e endpoint.Endpoint
	e = makeUppercaseProxy(u.String())
	e = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(e)
	e = kitratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(100, 100))(e) // 100 QPS per instance

	return e, nil
}

func makeUppercaseProxy(url string) endpoint.Endpoint {
	// TODO use a Client helper in transport/http
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(request); err != nil {
			return nil, fmt.Errorf("proxy: Encode: %v", err)
		}
		req, err := http.NewRequest("GET", url, &buf)
		if err != nil {
			return nil, fmt.Errorf("proxy: NewRequest: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("proxy: HTTP Client Do: %v", err)
		}
		defer resp.Body.Close()
		var response uppercaseResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("proxy: Decode: %v", err)
		}
		return response, nil
	}
}

func split(s string) []string {
	a := strings.Split(s, ",")
	for i := range a {
		a[i] = strings.TrimSpace(a[i])
	}
	return a
}
