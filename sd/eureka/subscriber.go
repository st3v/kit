package eureka

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/cache"
)

type subscriber struct {
	name   string
	client Client
	cache  *cache.Cache
	quitc  chan struct{}
}

var _ sd.Subscriber = new(subscriber)

func NewSubscriber(name string, client Client, factory sd.Factory, logger log.Logger) *subscriber {
	s := &subscriber{
		name:   name,
		client: client,
		cache:  cache.New(factory, logger),
		quitc:  make(chan struct{}),
	}

	s.cache.Update([]string{})

	go s.loop()

	return s
}

func (s *subscriber) loop() {
	entries := make(chan []string)
	defer close(entries)

	go s.client.WatchEntries(s.name, entries, s.quitc)
	for {
		select {
		case <-s.quitc:
			return
		case e := <-entries:
			s.cache.Update(e)
		}
	}
}

func (s *subscriber) Endpoints() ([]endpoint.Endpoint, error) {
	return s.cache.Endpoints(), nil
}

func (s *subscriber) Stop() {
	if s.quitc == nil {
		return
	}

	close(s.quitc)
	s.quitc = nil
}
