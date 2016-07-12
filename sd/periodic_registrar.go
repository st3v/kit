package sd

import (
	"fmt"
	"time"
)

type periodicRegistrar struct {
	registrar Registrar
	interval  time.Duration
	quitc     chan struct{}
}

func NewPeriodicRegistrar(r Registrar, interval time.Duration) Registrar {
	return &periodicRegistrar{
		registrar: r,
		interval:  interval,
	}
}

func (p *periodicRegistrar) Register() {
	if p.quitc != nil {
		return
	}

	p.quitc = make(chan struct{})

	go p.loop()
}

func (p *periodicRegistrar) loop() {
	for {
		select {
		case <-p.quitc:
			return
		case <-time.After(p.interval):
			fmt.Println("REG")
			p.registrar.Register()
		}
	}
}

func (p *periodicRegistrar) Deregister() {
	close(p.quitc)
	p.quitc = nil

	p.registrar.Deregister()
}
