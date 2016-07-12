package eureka

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hudl/fargo"

	"github.com/go-kit/kit/log"
)

const defaultPollInterval = 30 * time.Second

type Service struct {
	ID       string
	Name     string
	Address  string
	Port     int
	Metadata map[string]string
	TTL      time.Duration
}

type Client interface {
	GetEntries(name string) ([]string, error)
	WatchEntries(name string, entries chan []string, done chan struct{})
	Register(service *Service) error
	Deregister(service *Service) error
}

type fargoConnection interface {
	GetApp(name string) (*fargo.Application, error)
	RegisterInstance(*fargo.Instance) error
	DeregisterInstance(*fargo.Instance) error
	HeartBeatInstance(instance *fargo.Instance) error
	ScheduleAppUpdates(name string, await bool, done <-chan struct{}) <-chan fargo.AppUpdate
}

type client struct {
	conn fargoConnection
}

type clientConfig struct {
	httpClient   *http.Client
	pollInterval time.Duration
	logger       log.Logger
}

type Option func(*clientConfig)

func NewClient(addrs []string, options ...Option) Client {
	config := &clientConfig{
		pollInterval: defaultPollInterval,
	}

	for _, opt := range options {
		opt(config)
	}

	if config.httpClient != nil {
		fargo.HttpClient = config.httpClient
	}

	conn := fargo.NewConn(addrs...)
	conn.PollInterval = config.pollInterval

	return &client{&conn}
}

func HttpClient(client *http.Client) Option {
	return func(c *clientConfig) {
		c.httpClient = client
	}
}

func PollInterval(interval time.Duration) Option {
	return func(c *clientConfig) {
		c.pollInterval = interval
	}
}

func (c *client) GetEntries(name string) ([]string, error) {
	app, err := c.conn.GetApp(name)
	if err != nil {
		return nil, err
	}

	return appToEntries(app), nil
}

func (c *client) WatchEntries(name string, entries chan []string, done chan struct{}) {
	updates := c.conn.ScheduleAppUpdates(name, false, done)
	for {
		select {
		case <-done:
			return
		case u := <-updates:
			if u.Err != nil {
				entries <- []string{}
			} else {
				entries <- appToEntries(u.App)
			}
		}
	}
}

func (c *client) Register(service *Service) error {
	instance := serviceToInstance(service)

	if _, err := c.GetEntries(service.Name); err == nil {
		c.conn.HeartBeatInstance(instance)
		return nil
	}

	return c.conn.RegisterInstance(instance)
}

func (c *client) Deregister(service *Service) error {
	return c.conn.DeregisterInstance(serviceToInstance(service))
}

func appToEntries(app *fargo.Application) []string {
	entries := make([]string, len(app.Instances))
	for i, instance := range app.Instances {
		entries[i] = fmt.Sprintf("%s:%d", instance.HostName, instance.Port)
	}
	return entries
}

func serviceToInstance(s *Service) *fargo.Instance {
	instance := &fargo.Instance{
		InstanceID:       s.ID,
		App:              s.Name,
		HostName:         s.Address,
		IPAddr:           s.Address,
		VipAddress:       s.Address,
		SecureVipAddress: s.Address,
		Port:             s.Port,
		Status:           fargo.UP,
		DataCenterInfo:   fargo.DataCenterInfo{Name: fargo.MyOwn},
	}

	if s.TTL > 0 {
		instance.LeaseInfo = fargo.LeaseInfo{
			DurationInSecs:        int32(s.TTL.Seconds()),
			RenewalIntervalInSecs: int32(s.TTL.Seconds()),
		}
	}

	instance.SetMetadataString("instanceId", s.ID)

	if data, err := json.Marshal(s.Metadata); err == nil {
		instance.SetMetadataString("metadata", string(data))
	}

	return instance
}
