package http

import (
	"errors"

	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
)

// Client wraps a URL and provides a method that implements endpoint.Endpoint.
type Client struct {
	URL string
	context.Context
	DecodeFunc
	EncodeFunc
}

// Endpoint TODO
func (c Client) Endpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return nil, errors.New("not yet implemented")
	}
}
