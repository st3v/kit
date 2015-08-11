# Examples

TODO

1. [A minimal example](#a-minimal-example)
	1. [Your business logic](#your-business-logic)
	1. [Requests and responses](#requests-and-responses)
	1. [Endpoints](#endpoints)
	1. [Transports](#transports)
	1. [Test it out](#test-it-out)
1. [Logging and instrumentation](#logging-and-instrumentation)
	1. [Basic logging](#basic-logging)
	1. [Advanced logging](#advanced-logging)
	1. [Instrumentation](#instrumentation)
1. [Calling other services](#calling-other-services)
	1. [TODO](#todo)
	1. [TODO](#todo)
1. [Creating a client package](#creating-a-client-package)
	1. [Client-side endpoints](#client-side-endpoints)
	1. [Service discovery](#service-discovery)
	1. [Load balancing](#load-balancing)
1. [Request tracing](#request-tracing)
	1. [TODO](#todo)
	1. [TODO](#todo)

## A minimal example

Let's create a minimal Go kit service.

### Your business logic

Your service starts with your business logic.
In Go kit, we model a service as an **interface**.

```go
// StringService provides operations on strings.
type StringService interface {
	Uppercase(string) (string, error)
	Count(string) int
}
```

That interface will have an implementation.

```go
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
```

### Requests and responses

In Go kit, the primary messaging pattern is RPC.
So, every method in our interface will be modeled as a remote procedure call.
For each method, we define **request and response** structs,
 capturing all of the input and output parameters respectively.

```go
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
```

### Endpoints

Go kit provides much of its functionality through an abstraction called an **endpoint**.

```go
type Endpoint func(ctx context.Context, request interface{}) (response interface{}, err error)
```

An endpoint represents a single RPC.
That is, a single method in our service interface.
We'll write simple adapters to convert each of our service's methods into an endpoint.

```go
import (
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
```

### Transports

Now we need to expose your service to the outside world, so it can be called.
Your organization probably already has opinions about how services should talk to each other.
Maybe you use Thrift, or custom JSON over HTTP.
Go kit supports many **transports** out of the box.
(Adding support for new ones is easy—just [file an issue](https://github.com/go-kit/kit/issues).)

For this minimal example service, let's use JSON over HTTP.
Go kit provides a helper struct, in package transport/http.

```go
import (
	"encoding/json"
	"log"
	"net/http"

	"golang.org/x/net/context"

	httptransport "github.com/go-kit/kit/transport/http"
)

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

func decodeUppercaseRequest(r *http.Request) (interface{}, error) {
	var request uppercaseRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	if err := r.Body.Close(); err != nil {
		return nil, err
	}
	return request, nil
}

func decodeCountRequest(r *http.Request) (interface{}, error) {
	var request countRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	if err := r.Body.Close(); err != nil {
		return nil, err
	}
	return request, nil
}

func encodeResponse(w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}
```

### Test it out

The complete service is [stringsvc](https://github.com/go-kit/kit/blob/master/examples/stringsvc).

```
$ go get github.com/go-kit/kit/examples/stringsvc
$ stringsvc
```

```
$ curl -XPOST -d'{"s":"hello, world"}' localhost:8080/uppercase
{"v":"HELLO, WORLD","err":null}
$ curl -XPOST -d'{"s":"hello, world"}' localhost:8080/count
{"v":12}
```

## Logging and instrumentation

No service can be considered production-ready without thorough logging and instrumentation.
Go kit provides simple, robust, and extensible packages for both concerns.

### Basic logging

TODO

### Advanced logging

TODO

### Instrumentation

TODO

## Calling other services

TODO

## Creating a client package

TODO

### Client-side endpoints

TODO

### Service discovery

TODO

### Load balancing

TODO

## Request tracing

TODO
