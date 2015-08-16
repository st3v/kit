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

### stringsvc1

The complete service so far is [stringsvc1][].

[stringsvc1]: https://github.com/go-kit/kit/blob/master/examples/stringsvc1

```
$ go get github.com/go-kit/kit/examples/stringsvc1
$ stringsvc1
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

Logging is an important and primary concern in any microservice.
Any component that needs to log should treat the logger like a dependency, same as a database connection.
So, we construct our logger in our `func main`, and pass it to components that need it.
We never use a globally-scoped logger.

We could pass a logger directly to our application logic, but there's a better way.
Let's use a **middleware**, also known as decorator.
That's something that takes an endpoint and returns an endpoint.

```go
type Middleware func(Endpoint) Endpoint
```

Let's create a basic logging middleware.

```go
func loggingMiddleware(logger log.Logger) Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			logger.Log("msg", "calling endpoint")
			defer logger.Log("msg", "called endpoint")
			return next(ctx, request)
		}
	}
}
```

And wire it into each of our handlers.

```go
logger := log.NewLogfmtLogger(os.Stderr)

svc := stringService{}

var uppercase endpoint.Endpoint
uppercase = makeUppercaseEndpoint(svc)
uppercase = loggingMiddleware(log.NewContext(logger, "method", "uppercase"))(uppercase)

var count endpoint.Endpoint
count = makeCountEndpoint(svc)
count = loggingMiddleware(log.NewContext(logger, "method", "count"))(count)

uppercaseHandler := httptransport.Server{
	Endpoint: uppercase,
	// ...
}

countHandler := httptransport.Server{
	Endpoint: count,
	// ...
}
```

It turns out that this technique is useful for a lot more than just logging.
Most Go kit components are implemented as endpoint middlewares.

### Advanced logging

But what if we want to log in our application domain, like the parameters that are passed in?
We can define a middleware for our service, and get the same, nice, composable behavior.
Since our StringService is defined as an interface, we just need to make a new type
 which wraps an existing StringService, and performs the extra logging duties.

```go
type loggingMiddleware struct{
	logger log.Logger
	StringService
}


func (mw loggingMiddleware) Uppercase(s string) (output string, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "uppercase",
			"input", s,
			"output", output,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	output, err = mw.StringService.Uppercase(s)
	return
}

func (mw loggingMiddleware) Count(s string) (n int) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "count",
			"input", s,
			"n", n,
			"took", time.Since(begin),
		)
	}(time.Now())

	n = mw.StringService.Count(s)
	return
}
```

And wire it in.

```go
import (
	"os"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
)

func main() {
	logger := log.NewLogfmtLogger(os.Stderr)

	svc := stringService{}
	svc = loggingMiddleware{logger, svc}

	uppercaseHandler := httptransport.Server{
		Endpoint: makeUppercaseEndpoint(svc),
		// ...
	}

	countHandler := httptransport.Server{
		Endpoint: makeCountEndpoint(svc),
		// ...
	}
}
```

Use endpoint middlewares for transport-domain concerns, like circuit breaking and rate limiting.
Use service middlewares for business-domain concerns, like logging and instrumentation.
Speaking of instrumentation...

### Instrumentation

Proper instrumentation is just as important as logging.
But what is instrumentation? There are potentially several definitions.
In Go kit, instrumentation means using **package metrics** to record statistics about your service's runtime behavior.
Counting the number of jobs processed,
 recording the duration of requests after they've finished,
  and tracking the number of in-flight operations would all be considered instrumentation.

We can use the same middleware pattern that we used for logging.

```go
type instrumentingMiddleware struct {
	requestCount   metrics.Counter
	requestLatency metrics.TimeHistogram
	countResult    metrics.Histogram
	StringService
}

func (mw instrumentingMiddleware) Uppercase(s string) (output string, err error) {
	defer func(begin time.Time) {
		methodField := metrics.Field{Key: "method", Value: "uppercase"}
		errorField := metrics.Field{Key: "error", Value: fmt.Sprintf("%v", err)}
		mw.requestCount.With(methodField).With(errorField).Add(1)
		mw.requestLatency.With(methodField).With(errorField).Observe(time.Since(begin))
	}(time.Now())

	output, err = mw.StringService.Uppercase(s)
	return
}

func (mw instrumentingMiddleware) Count(s string) (n int) {
	defer func(begin time.Time) {
		methodField := metrics.Field{Key: "method", Value: "count"}
		errorField := metrics.Field{Key: "error", Value: fmt.Sprintf("%v", error(nil))}
		mw.requestCount.With(methodField).With(errorField).Add(1)
		mw.requestLatency.With(methodField).With(errorField).Observe(time.Since(begin))
		mw.countResult.Observe(int64(n))
	}(time.Now())

	n = mw.StringService.Count(s)
	return
}
```

And wire it into our service.

```go
import (
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-kit/kit/metrics"
)

func main() {
	logger := log.NewLogfmtLogger(os.Stderr)

	fieldKeys := []string{"method", "error"}
	requestCount := kitprometheus.NewCounter(stdprometheus.CounterOpts{
		// ...
	}, fieldKeys)

	requestLatency := metrics.NewTimeHistogram(time.Microsecond, kitprometheus.NewSummary(stdprometheus.SummaryOpts{
		// ...
	}, fieldKeys))

	countResult := kitprometheus.NewSummary(stdprometheus.SummaryOpts{
		// ...
	}, []string{}))

	svc := stringService{}
	svc = loggingMiddleware(logger, svc)
	svc = instrumentingMiddleware{requestCount, requestLatency, countResult, svc}

	uppercaseHandler := httptransport.Server{
		Endpoint: makeUppercaseEndpoint(svc),
		// ...
	}

	countHandler := httptransport.Server{
		Endpoint: makeCountEndpoint(svc),
		// ...
	}
}
```

### stringsvc2

The complete service so far is [stringsvc2][].

[stringsvc2]: https://github.com/go-kit/kit/blob/master/examples/stringsvc2

```
$ go get github.com/go-kit/kit/examples/stringsvc2
$ stringsvc2
msg=HTTP addr=:8080
```

```
$ curl -XPOST -d'{"s":"hello, world"}' localhost:8080/uppercase
{"v":"HELLO, WORLD","err":null}
$ curl -XPOST -d'{"s":"hello, world"}' localhost:8080/count
{"v":12}
```

```
method=uppercase input="hello, world" output="HELLO, WORLD" err=null took=2.455µs
method=count input="hello, world" n=12 took=743ns
```

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
