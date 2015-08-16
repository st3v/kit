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
		if proxyList == "" {
			_ = logger.Log("proxy", "none")
			return next
		}

		instances := split(proxyList)
		_ = logger.Log("proxy", fmt.Sprint(instances))

		var (
			publisher   = static.NewPublisher(instances, factory, logger) // could use any Publisher here
			lb          = loadbalancer.NewRoundRobin(publisher)
			maxAttempts = 3
			maxTime     = 100 * time.Millisecond
			endpoint    = loadbalancer.Retry(maxAttempts, maxTime, lb)
		)
		return proxymw{ctx, endpoint, next}
	}
}

type proxymw struct {
	context.Context
	UppercaseEndpoint endpoint.Endpoint
	StringService
}

func (mw proxymw) Uppercase(s string) (string, error) {
	response, err := mw.UppercaseEndpoint(mw.Context, uppercaseRequest{S: s})
	if err != nil {
		return "", err
	}
	resp := response.(uppercaseResponse)
	return resp.V, resp.Err
}

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

	var e endpoint.Endpoint
	e = makeUppercaseProxy(u.String())
	e = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(e)   // circuit breaker on each individual instance
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
