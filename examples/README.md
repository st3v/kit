# Examples

TODO

1. [A minimal example](#a-minimal-example)
	1. [Your business logic](#your-business-logic)
	1. [Requests and responses](#requests-and-responses)
	1. [Endpoints](#endpoints)
	1. [Transports](#transports)
	1. [stringsvc1](#stringsvc1)
1. [Logging and instrumentation](#logging-and-instrumentation)
	1. [Transport logging](#transport-logging)
	1. [Application logging](#application-logging)
	1. [Instrumentation](#instrumentation)
	1. [stringsvc2](#stringsvc2)
1. [Calling other services](#calling-other-services)
	1. [Client-side endpoints and middleware](#client-side-endpoints-and-middleware)
	1. [Service discovery and load balancing](#service-discovery-and-load-balancing)
	1. [Using a service middleware](#using-a-service-middleware)
	1. [stringsvc3](#stringsvc3)
1. [Advanced topics](#advanced-topics)
	1. [Creating a client package](#creating-a-client-package)
	1. [Request tracing](#request-tracing)
	1. [Threading a context](#threading-a-context)
1. [The final product](#the-final-product)

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

### Transport logging

Any component that needs to log should treat the logger like a dependency, same as a database connection.
So, we construct our logger in our `func main`, and pass it to components that need it.
We never use a globally-scoped logger.

We could pass a logger directly into our stringService implementation, but there's a better way.
Let's use a **middleware**, also known as decorator.
A middleware is a function that takes an endpoint and returns an endpoint.

```go
type Middleware func(Endpoint) Endpoint
```

In between, it can do anything.
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
uppercase = loggingMiddleware(log.NewContext(logger).With("method", "uppercase"))(uppercase)

var count endpoint.Endpoint
count = makeCountEndpoint(svc)
count = loggingMiddleware(log.NewContext(logger).With("method", "count"))(count)

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
Many Go kit components are implemented as endpoint middlewares.

### Application logging

But what if we want to log in our application domain, like the parameters that are passed in?
It turns out that we can define a middleware for our service, and get the same nice and composable effects.
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
	svc = loggingMiddleware{logger, svc}
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

It's rare that a service exists in a vacuum.
Often, you need to call other services.
**This is where Go kit shines**.
We provide transport middlewares to solve many of the problems that come up.
Let's provide a commandline flag to our stringsvc to proxy uppercase requests somewher else.

```go
func main() {
	var (
		listen = flag.String("listen", ":8080", "HTTP listen address")
		proxy  = flag.String("proxy", "", "Optional URL to proxy uppercase requests")
	)
	flag.Parse()

	// ...

	var uppercase endpoint.Endpoint
	if *proxy != "" {
		uppercase = makeUppercaseProxy(*proxy)
	} else {
		uppercase = makeUppercaseEndpoint(svc)
	}

	// ...
}
```

The makeUppercaseProxy function just converts a URL to an endpoint.

```go
func makeUppercaseProxy(url string) endpoint.Endpoint {
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
```

We've created a client endpoint.
It's exactly the same _type_ as a server endpoint, but we use it to invoke, rather than serve, a request.
That symmetry is nice: it allows us to reuse the same set of value-add middlewares.
And that's important, because calling a remote service over the network isn't the same as invoking a method on a local object.
There are lots of failure modes we need to account for.

### Client-side endpoints and middleware

Let's add rate limiting and circuit breaking to the client endpoint.

```go
import (
	jujuratelimit "github.com/juju/ratelimit"
	"github.com/sony/gobreaker"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	kitratelimit "github.com/go-kit/kit/ratelimit"
)

func main() {
	// ...

	var uppercase endpoint.Endpoint
	if *proxy != "" {
		uppercase = makeUppercaseProxy(*proxy)
		uppercase = circuitbreaker.NewGobreaker(gobreaker.NewBreaker(gobreaker.Settings{}))(uppercase)
		uppercase = kitratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(100, 100))(uppercase) // 100 QPS
	} else {
		uppercase = makeUppercaseEndpoint(svc)
	}

	// ...
}
```

Go kit provides a helper method to chain middlewares like this.
Note that the application order is reversed.
(Also note that it's important to wrap the circuit breaker with the rate limiter, and not the other way around.)

```go
uppercase = makeUppercaseProxy(*proxy)
uppercase = endpoint.Chain(
	kitratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(100, 100)), // 100 QPS
	circuitbreaker.NewGobreaker(gobreaker.NewBreaker(gobreaker.Settings{})),
)(uppercase)
```

### Service discovery and load balancing

What we've built so far is fine, as long as the proxying service has a single fixed URL.
But in reality, we'll probably have a set of stateless service instances to choose from.
And they'll probably be dynamic, constantly changing as instances come up and go down.
This is the realm of service discovery.
And Go kit provides adapters to service discovery systems.

How to construct those adapters differs depending on the specifics of the system.
But they all implement the same [loadbalancer.Publisher][publisher] interface.
From there, we can wrap them with one of several [loadbalancer.LoadBalancer][loadbalancer] implementations, like random or round-robin.
Finally, a [loadbalancer.Retry][retry] converts the load balancer to a usable client endpoint.

[publisher]: https://godoc.org/github.com/go-kit/kit/loadbalancer#Publisher
[loadBalancer]: https://godoc.org/github.com/go-kit/kit/loadbalancer#LoadBalancer
[retry]: https://godoc.org/github.com/go-kit/kit/loadbalancer#Retry

```go
name := "mysvc.internal.net"
ttl := 5 * time.Second
publisher := dnssrv.NewPublisher(name, ttl, factory, logger) // could use any Publisher here
lb := loadbalancer.NewRoundRobin(publisher)
maxAttempts := 3
maxTime := 100*time.Millisecond
clientEndpoint := loadbalancer.Retry(maxAttempts, maxTime, lb)
```

The factory is a function that converts an instance string (typically a host:port) to an endpoint.

```go
func factory(instance string) (endpoint.Endpoint, error) {
	return makeUppercaseProxy("http://" + instance + "/uppercase"), nil
}
```

But that's not enough.
At a minimum, we should add a circuit breaker, and perhaps rate limiting.
And we should add them to each individual endpoint, rather than the aggregate load-balanced endpoint.

```go
func factory(instance string) (endpoint.Endpoint, error) {
	var e endpoint.Endpoint
	e = makeUppercaseProxy("http://" + instance + "/uppercase")
	e = circuitbreaker.NewGobreaker(gobreaker.NewBreaker(gobreaker.Settings{}))(e)
	e = kitratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(100, 100))(e) // 100 QPS
	return e
}
```

### Using a service middleware

Of course, we'd like to log and instrument the requests we forward to the remote service.
We can't do that effectively in the endpoint domain, as we don't have access to request parameters.
We need to be in the service domain for that.
So, let's write a service middleware, to proxy Uppercase requests.

```go
type proxyingMiddleware struct {
	context.Context
	UppercaseEndpoint endpoint.Endpoint // load-balanced
	StringService                       // for all other requests i.e. Count
}

func (mw proxyingMiddleware) Uppercase(s string) (string, error) {
	response, err := mw.UppercaseEndpoint(mw.Context, uppercaseRequest{S: s})
	if err != nil {
		return "", err
	}
	resp := response.(uppercaseResponse)
	return resp.V, resp.Err
}
```

If the user specifies the -proxy flag, we can wire it in to the component graph.

```go
var svc StringService
svc = stringService{}
if *proxy != "" {
	svc = proxyingMiddleware(*proxy, ctx, logger)(svc)
	_ = logger.Log("proxy", *proxy)
} else {
	_ = logger.Log("proxy", "none")
}
svc = loggingMiddleware(logger)(svc)
svc = instrumentingMiddleware(requestCount, requestLatency, countResult)(svc)
```

### stringsvc3

The complete service so far is [stringsvc3][].

[stringsvc3]: https://github.com/go-kit/kit/blob/master/examples/stringsvc3

```
$ go get github.com/go-kit/kit/examples/stringsvc3
$ stringsvc3 -listen=:8001 &
listen=:8001 proxy=none
listen=:8001 msg=HTTP addr=:8001
$ stringsvc3 -listen=:8002 -proxy=localhost:8001
listen=:8002 proxy=localhost:8001
listen=:8002 msg=HTTP addr=:8002
```

```
$ curl -XPOST -d'{"s":"hello, world"}' localhost:8002/uppercase
{"v":"HELLO, WORLD","err":null}
```

```
listen=:8001 method=uppercase input="hello, world" output="HELLO, WORLD" err=null took=2.119µs
listen=:8002 method=uppercase input="hello, world" output="HELLO, WORLD" err=null took=18.119568ms
```

## Advanced topics

### Request tracing

Once your infrastructure grows beyond a certain size, it becomes important to trace requests through multiple services, so you can identify and troubleshoot hotspots.
See [package tracing](https://github.com/go-kit/kit/tracing) for more information.

### Creating a client package

It's possible to use Go kit to create a client package to your service, to make consuming your service easier from other Go programs.
Effectively, your client package will provide an implementation of your service interface, which invokes a remote service instance using a specific transport.
An example is in the works. Stay tuned.

### Threading a context

The context object is used to carry information across conceptual boundaries in the scope of a single request.
In our example, we haven't threaded the context through our business logic.
But it may be useful to do so, to get access to request-scoped information like trace IDs.
It may also be possible to pass things like loggers and metrics objects through the context, although this is not currently recommended.
