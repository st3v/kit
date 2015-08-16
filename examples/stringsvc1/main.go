package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
)

// StringService provides operations on strings.
type StringService interface {
	Uppercase(string) (string, error)
	Count(string) int
}

type stringService struct{}

func (stringService) Uppercase(s string) (string, error) {
	if s == "" {
		return "", ErrEmpty
	}
	return strings.ToUpper(s), nil
}

func (stringService) Count(s string) int {
	return len(s)
}

func main() {
	ctx := context.Background()
	svc := stringService{}

	uppercaseHandler := httptransport.Server{
		Context:    ctx,
		Endpoint:   makeUppercaseEndpoint(svc),
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
	log.Fatal(http.ListenAndServe(":8080", nil))
}

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

// ErrEmpty is returned when an input string is empty.
var ErrEmpty = errors.New("empty string")
