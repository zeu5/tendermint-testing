package consensus

import (
	"sync"

	"github.com/ImperiumProject/go-replicaclient"
	"github.com/tendermint/tendermint/libs/service"
)

type controlledTicker struct {
	controller     *replicaclient.ReplicaClient
	fromController chan replicaclient.TimeoutInfo
	outChan        chan timeoutInfo
	service.BaseService
	ti   timeoutInfo
	lock *sync.Mutex
}

var _ TimeoutTicker = &controlledTicker{}

func NewControlledTimeoutTicker() *controlledTicker {
	controller, _ := replicaclient.GetClient()
	t := &controlledTicker{
		controller:     controller,
		fromController: controller.TimeoutChan(),
		outChan:        make(chan timeoutInfo, 10),
		lock:           new(sync.Mutex),
	}
	t.BaseService = *service.NewBaseService(nil, "ControlledTimer", t)
	return t
}

func (c *controlledTicker) Chan() <-chan timeoutInfo {
	return c.outChan
}

func (c *controlledTicker) ScheduleTimeout(newti timeoutInfo) {
	c.lock.Lock()
	ti := c.ti
	c.lock.Unlock()

	if newti.Height < ti.Height {
		return
	} else if newti.Height == ti.Height {
		if newti.Round < ti.Round {
			return
		} else if newti.Round == ti.Round {
			if ti.Step > 0 && newti.Step <= ti.Step {
				return
			}
		}
	}
	c.lock.Lock()
	c.ti = newti
	c.lock.Unlock()
	c.controller.StartTimer(newti)
}

func (c *controlledTicker) OnStart() error {
	if err := c.controller.Start(); err != nil {
		return err
	}
	go c.poll()
	return nil
}

func (c *controlledTicker) poll() {
	for {
		select {
		case info := <-c.fromController:
			ti, ok := info.(timeoutInfo)
			if ok {
				c.lock.Lock()
				ok := c.ti.Key() == ti.Key()
				c.lock.Unlock()
				if ok {
					c.outChan <- ti
				}
			}
		case <-c.Quit():
			return
		}
	}
}
