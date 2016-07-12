package eureka_test

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/eureka"
	"github.com/go-kit/kit/sd/eureka/mock"
)

func TestSubscriber(t *testing.T) {
	var (
		entriesChan = make(chan []string, 1)
		doneChan    = make(chan struct{})
		client      = new(mock.Client)
	)

	client.WatchEntriesStub = func(name string, entries chan []string, done chan struct{}) {
		go func() {
			for {
				select {
				case <-done:
					close(doneChan)
					return
				case e := <-entriesChan:
					entries <- e
				}
			}
		}()
	}

	subscriber := eureka.NewSubscriber("foo.bar.baz", client, testFactory, log.NewNopLogger())
	defer subscriber.Stop()

	// start empty
	eventually(assertEndpoints(subscriber, []string{}), t.Error, 1*time.Second, 5*time.Millisecond)

	// add instances
	entries := []string{"instance-1", "instance-2"}
	entriesChan <- entries
	eventually(assertEndpoints(subscriber, entries), t.Error, 1*time.Second, 5*time.Millisecond)

	// remove one add another
	entries = []string{"instance-1", "instance-3"}
	entriesChan <- entries
	eventually(assertEndpoints(subscriber, entries), t.Error, 1*time.Second, 5*time.Millisecond)

	// remove all instances
	entries = []string{}
	entriesChan <- []string{}
	eventually(assertEndpoints(subscriber, entries), t.Error, 1*time.Second, 5*time.Millisecond)

	// stop subscriber
	subscriber.Stop()
	eventually(assertClosedChannel(doneChan), t.Error, 1*time.Second, 5*time.Millisecond)
}

type assertion func() error

func eventually(assert assertion, report func(...interface{}), timeout, interval time.Duration) {
	var err error

	cancel := time.After(timeout)

	for {
		select {
		case <-cancel:
			report(err)
			return
		case <-time.After(interval):
			if err = assert(); err == nil {
				return
			}
		}
	}
}

func assertEndpoints(s sd.Subscriber, expected []string) assertion {
	return func() error {
		endpoints, err := s.Endpoints()
		if err != nil {
			return err
		}

		// if have, want := len(endpoints), len(expected); have != want {
		// 	return fmt.Errorf("Endpoints len: want %d, have %d", want, have)
		// }

		actual := make([]string, 0, len(endpoints))
		for _, e := range endpoints {
			resp, err := e(context.Background(), struct{}{})
			if err != nil {
				return err
			}

			actual = append(actual, resp.(string))
		}

		if have, want := sortAndJoin(actual), sortAndJoin(expected); have != want {
			return fmt.Errorf("Endpoint instances: want %q, have %q", want, have)
		}

		return nil
	}
}

func assertClosedChannel(c chan struct{}) assertion {
	return func() error {
		select {
		case <-c:
			return nil
		default:
			return errors.New("channel not closed")
		}
	}
}

func sortAndJoin(a []string) string {
	sort.Strings(a)
	return strings.Join(a, ", ")
}

func testFactory(instance string) (endpoint.Endpoint, io.Closer, error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return instance, nil
	}, nil, nil
}
