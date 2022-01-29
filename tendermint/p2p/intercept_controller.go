package p2p

import (
	"encoding/json"
	"strconv"

	"github.com/ImperiumProject/go-replicaclient"
	"github.com/ImperiumProject/imperium/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/service"
)

type Controller interface {
	SendChan() chan ControllerMsgEnvelop
	ReceiveChan() chan ControllerMsgEnvelop
	service.Service
}

type ControllerMsgEnvelop struct {
	ChannelID ChannelID `json:"chan_id"`
	MsgB      []byte    `json:"msg"`
	From      NodeID    `json:"from"`
	To        NodeID    `json:"to"`
}

type DummyController struct {
	service.BaseService
	inChan   chan ControllerMsgEnvelop
	outChan  chan ControllerMsgEnvelop
	stopchan chan bool
}

func NewDummyController(logger log.Logger, buf int) *DummyController {
	c := &DummyController{
		inChan:   make(chan ControllerMsgEnvelop, buf),
		outChan:  make(chan ControllerMsgEnvelop, buf),
		stopchan: make(chan bool, 1),
	}
	c.BaseService = *service.NewBaseService(logger, "DummyController", c)
	return c
}

func (d *DummyController) SendChan() chan ControllerMsgEnvelop {
	return d.inChan
}

func (d *DummyController) ReceiveChan() chan ControllerMsgEnvelop {
	return d.outChan
}

func (d *DummyController) poll() {
	for {
		select {
		case msg := <-d.inChan:
			// Should not be doing this. The same replica is going to get a message it sent.
			// Logic for figuring out what to do with the message goes here!
			d.outChan <- msg
		case <-d.stopchan:
			return
		}
	}
}

func (d *DummyController) OnStart() error {
	go d.poll()
	return nil
}

func (d *DummyController) OnStop() {
	d.stopchan <- true
}

func (d *DummyController) OnReset() error {
	return nil
}

type MasterControllerConfig struct {
	MasterAddr string
	ListenAddr string
	ExternAddr string
	NodeID     NodeID
	NodeInfo   map[string]interface{}
	BufSize    int
}

type MasterController struct {
	service.BaseService
	conf             MasterControllerConfig
	directiveHandler replicaclient.DirectiveHandler
	controller       *replicaclient.ReplicaClient
	outChan          chan ControllerMsgEnvelop
	inChan           chan ControllerMsgEnvelop
	stopCh           chan bool
}

func NewMasterController(
	conf MasterControllerConfig,
	directiveHandler replicaclient.DirectiveHandler,
	logger log.Logger,
) *MasterController {
	replicaclient.Init(
		&replicaclient.Config{
			ReplicaID:        types.ReplicaID(conf.NodeID),
			ImperiumAddr:     conf.MasterAddr,
			ClientServerAddr: conf.ListenAddr,
			ClientAdvAddr:    conf.ExternAddr,
			Info:             conf.NodeInfo,
		},
		directiveHandler,
		logger,
	)
	controller, _ := replicaclient.GetClient()
	c := &MasterController{
		conf:             conf,
		directiveHandler: directiveHandler,
		controller:       controller,
		outChan:          make(chan ControllerMsgEnvelop, conf.BufSize),
		inChan:           make(chan ControllerMsgEnvelop, conf.BufSize),
		stopCh:           make(chan bool),
	}

	c.BaseService = *service.NewBaseService(logger, "MasterController", c)
	return c
}

func (c *MasterController) ReceiveChan() chan ControllerMsgEnvelop {
	return c.outChan
}

func (c *MasterController) SendChan() chan ControllerMsgEnvelop {
	return c.inChan
}

func (c *MasterController) OnStart() error {
	err := c.controller.Start()
	if err != nil {
		return err
	}
	c.controller.Ready()
	go c.poll()
	return nil
}

func (c *MasterController) OnStop() {
	c.controller.Stop()
	close(c.stopCh)
}

func (c *MasterController) OnReset() error {
	return nil
}

func (c *MasterController) handleIncoming(m ControllerMsgEnvelop) {
	m.From = c.conf.NodeID
	msgB, err := json.Marshal(m)
	if err != nil {
		return
	}
	// c.Logger.Debug("Controller: sending message", "msg", string(msgB))
	c.controller.SendMessage(
		strconv.Itoa(int(m.ChannelID)),
		types.ReplicaID(m.To),
		msgB,
		true,
	)
}

func (c *MasterController) handleOutgoing(m *types.Message) {
	msg := ControllerMsgEnvelop{}
	err := json.Unmarshal([]byte(m.Data), &msg)
	if err != nil {
		return
	}
	// c.Logger.Debug("Controller: received message", "msg", m.Msg)
	msg.From = NodeID(m.From)
	msg.To = NodeID(m.To)
	c.outChan <- msg
}

func (c *MasterController) poll() {
	for {
		select {
		case m := <-c.inChan:
			go c.handleIncoming(m)
		case <-c.stopCh:
			return
		default:
			m, ok := c.controller.ReceiveMessage()
			if ok {
				go c.handleOutgoing(m)
			}
		}
	}
}
