package eureka_test

import (
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd/eureka"
	"github.com/go-kit/kit/sd/eureka/mock"
)

func TestRegistrar(t *testing.T) {
	service := &eureka.Service{}
	client := new(mock.Client)

	registrar := eureka.NewRegistrar(service, client, log.NewNopLogger())

	testcases := []struct {
		fn            func()
		fname         string
		haveCallCount func() int
		wantCallCount int
		haveArg       func(int) *eureka.Service
		wantArg       *eureka.Service
	}{
		{registrar.Register, "client.Register", client.RegisterCallCount, 1, client.RegisterArgsForCall, service},
		{registrar.Deregister, "client.Deregister", client.DeregisterCallCount, 1, client.DeregisterArgsForCall, service},
	}

	for _, test := range testcases {
		test.fn()

		if have, want := test.haveCallCount(), test.wantCallCount; have != want {
			t.Fatalf("%s call count: want %d, have %d", test.fname, want, have)
		}

		lastCall := test.wantCallCount - 1
		if lastCall < 0 {
			continue
		}

		if have, want := test.haveArg(lastCall), test.wantArg; have != want {
			t.Fatalf("%s call args: want %v, have %v", test.fname, want, have)
		}
	}
}
