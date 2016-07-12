// +build integration

package eureka_test

import (
	"os"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd/eureka"
)

func TestIntegration(t *testing.T) {
	// docker run -p 8080:8080 netflixoss/eureka:1.3.1
	// export EUREKA_URL=http://localhost:8080/eureka

	addr := os.Getenv("EUREKA_URL")
	if addr == "" {
		t.Fatal("EUREKA_URL is not set")
	}

	client := eureka.NewClient([]string{addr}, eureka.PollInterval(1*time.Second))
	logger := log.NewLogfmtLogger(os.Stderr)

	service := &eureka.Service{
		ID:       "id",
		Name:     "foo.bar.baz",
		Address:  "example.com",
		Port:     1234,
		Metadata: map[string]string{"foo": "bar"},
		TTL:      900 * time.Second,
	}

	subscriber := eureka.NewSubscriber(service.Name, client, testFactory, logger)

	eventually(
		assertEndpoints(subscriber, []string{}),
		t.Error,
		1*time.Second,
		100*time.Millisecond,
	)

	registrar := eureka.NewRegistrar(
		service,
		client,
		log.NewContext(logger).With("component", "registrar"),
	)

	registrar.Register()
	defer registrar.Deregister()

	eventually(
		assertEndpoints(subscriber, []string{"example.com:1234"}),
		t.Error,
		1*time.Minute,
		100*time.Millisecond,
	)

	registrar.Deregister()

	eventually(
		assertEndpoints(subscriber, []string{}),
		t.Error,
		1*time.Minute,
		100*time.Millisecond,
	)
}
