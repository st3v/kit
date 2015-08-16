package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	jujuratelimit "github.com/juju/ratelimit"
	"github.com/sony/gobreaker"
	"golang.org/x/net/context"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	kitratelimit "github.com/go-kit/kit/ratelimit"
)

func factory(instance string) (endpoint.Endpoint, error) {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}

	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		req, err := http.NewRequest("GET", url, &buf)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		var response uppercaseResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, err
		}
		return response, nil
	}
}
