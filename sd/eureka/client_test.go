package eureka

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/hudl/fargo"
)

func TestRegister(t *testing.T) {
	var (
		service  = testService("name", "address", 0)
		instance = serviceToInstance(service)
		conn     = newMockConn()
		client   = &client{conn}
	)

	// initial register
	if err := client.Register(service); err != nil {
		t.Fatal(err)
	}

	if have, want := len(conn.registry), 1; have != want {
		t.Fatalf("want %d, have %d", want, have)
	}

	if have, want := conn.registry[0].instance, instance; !reflect.DeepEqual(have, want) {
		t.Errorf("want %+v, have %+v", want, have)
	}

	if have, want := conn.registry[0].heartbeats, 0; have != want {
		t.Errorf("want %d, have %d", want, have)
	}

	// subsequent register
	if err := client.Register(service); err != nil {
		t.Fatal(err)
	}

	if have, want := conn.registry[0].heartbeats, 1; have != want {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestDeregister(t *testing.T) {
	var (
		service = testService("name", "address", 0)
		conn    = newMockConn()
		client  = &client{conn}
	)

	conn.RegisterInstance(serviceToInstance(service))

	if err := client.Deregister(service); err != nil {
		t.Fatal(err)
	}

	if have, want := len(conn.registry), 0; have != want {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestGetEntries(t *testing.T) {
	var (
		services = []*Service{
			testService("name", "a", 123),
			testService("name", "b", 987),
		}

		conn   = newMockConn()
		client = &client{conn}
	)

	for _, s := range services {
		conn.RegisterInstance(serviceToInstance(s))
	}

	entries, err := client.GetEntries("name")
	if err != nil {
		t.Fatal(err)
	}

	sort.Strings(entries)
	if have, want := strings.Join(entries, ", "), "a:123, b:987"; have != want {
		t.Fatalf("want %q, have %q", want, have)
	}
}

func TestWatchEntries(t *testing.T) {
	var (
		conn    = newMockConn()
		client  = &client{conn}
		updates = make(chan []string)
		done    = make(chan struct{})
	)
	defer close(done)

	go client.WatchEntries("name", updates, done)

	go func() {
		select {
		case <-time.After(1 * time.Second):
			t.Fatal("timed out")
		case entries := <-updates:
			sort.Strings(entries)
			if have, want := strings.Join(entries, ", "), "a:123, b:987"; have != want {
				t.Fatalf("want %q, have %q", want, have)
			}
		}
	}()

	conn.updates <- fargo.AppUpdate{
		App: &fargo.Application{
			Instances: []*fargo.Instance{
				serviceToInstance(testService("name", "a", 123)),
				serviceToInstance(testService("name", "b", 987)),
			},
		},
		Err: nil,
	}
}

func TestServiceToInstance(t *testing.T) {}

func testService(name string, addr string, port int) *Service {
	return &Service{
		ID:       "id",
		Name:     name,
		Address:  addr,
		Port:     port,
		Metadata: map[string]string{"key": "value"},
		TTL:      987 * time.Second,
	}
}

type registryEntry struct {
	instance   *fargo.Instance
	heartbeats int
}

type mockConn struct {
	registry []*registryEntry
	updates  chan fargo.AppUpdate
}

func newMockConn() *mockConn {
	return &mockConn{
		registry: []*registryEntry{},
		updates:  make(chan fargo.AppUpdate, 1),
	}
}

func (mc *mockConn) entry(instance *fargo.Instance) (int, *registryEntry) {
	for i, entry := range mc.registry {
		if entry.instance.HostName == instance.HostName {
			return i, entry
		}
	}
	return -1, nil
}

func (mc *mockConn) instances(name string) []*fargo.Instance {
	instances := []*fargo.Instance{}
	for _, entry := range mc.registry {
		if entry.instance.App == name {
			instances = append(instances, entry.instance)
		}
	}
	return instances
}

func (mc *mockConn) GetApp(name string) (*fargo.Application, error) {
	if instances := mc.instances(name); len(instances) == 0 {
		return nil, errors.New("App not registered")
	} else {
		return &fargo.Application{Instances: instances}, nil
	}
}

func (mc *mockConn) RegisterInstance(i *fargo.Instance) error {
	if idx, _ := mc.entry(i); idx >= 0 {
		return errors.New("Instance already registerd")
	}

	mc.registry = append(mc.registry, &registryEntry{i, 0})
	return nil
}

func (mc *mockConn) DeregisterInstance(i *fargo.Instance) error {
	if idx, _ := mc.entry(i); idx < 0 {
		return errors.New("Instance not registered")
	} else {
		mc.registry = append(mc.registry[:idx], mc.registry[idx+1:]...)
		return nil
	}
}

func (mc *mockConn) HeartBeatInstance(i *fargo.Instance) error {
	if idx, e := mc.entry(i); idx < 0 {
		return errors.New("Instance not registered")
	} else {
		e.heartbeats++
		return nil
	}
}

func (mc *mockConn) ScheduleAppUpdates(name string, await bool, done <-chan struct{}) <-chan fargo.AppUpdate {
	return mc.updates
}
