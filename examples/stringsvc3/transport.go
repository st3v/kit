package main

import (
	"encoding/json"
	"io"

	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
)

func makeUppercaseEndpoint(svc StringService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(uppercaseRequest)
		v, err := svc.Uppercase(req.S)
		return uppercaseResponse{v, err}, nil
	}
}

func makeCountEndpoint(svc StringService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(countRequest)
		v := svc.Count(req.S)
		return countResponse{v}, nil
	}
}

func decodeUppercaseRequest(r io.Reader) (interface{}, error) {
	var request uppercaseRequest
	if err := json.NewDecoder(r).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func decodeCountRequest(r io.Reader) (interface{}, error) {
	var request countRequest
	if err := json.NewDecoder(r).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func encodeResponse(w io.Writer, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}

type uppercaseRequest struct {
	S string `json:"s"`
}

type uppercaseResponse struct {
	V   string `json:"v"`
	Err error  `json:"err"`
}

type countRequest struct {
	S string `json:"s"`
}

type countResponse struct {
	V int `json:"v"`
}
