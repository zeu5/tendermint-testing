package p2p

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/service"
	"github.com/tendermint/tendermint/p2p/conn"
)

type InterceptedTransportOptions struct {
	MConnOptions MConnTransportOptions
}

type router struct {
	service.BaseService
	inChan      chan ControllerMsgEnvelop
	connections map[NodeID]*InterceptedConnection
	lock        *sync.Mutex
	stopChan    chan bool
}

func newRouter(logger log.Logger, inChan chan ControllerMsgEnvelop) *router {
	r := &router{
		connections: make(map[NodeID]*InterceptedConnection),
		inChan:      inChan,
		lock:        new(sync.Mutex),
		stopChan:    make(chan bool, 1),
	}
	r.BaseService = *service.NewBaseService(logger, "InterceptRouter", r)
	return r
}

func (r *router) AddConnection(nodeId NodeID, conn *InterceptedConnection) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.connections[nodeId] = conn
}

func (r *router) OnStop() {
	r.lock.Lock()
	for _, c := range r.connections {
		c.Close()
	}
	r.lock.Unlock()
	r.stopChan <- true
}

func (r *router) sendMsg(msgE ControllerMsgEnvelop) {

	r.lock.Lock()
	conn, ok := r.connections[msgE.From]
	r.lock.Unlock()

	if ok {
		// r.Logger.Info("Sending message to connection", "msg", msgE)
		conn.inChan <- msgE
	}
}

func (r *router) poll() {
	for {
		select {
		case msgE := <-r.inChan:
			go r.sendMsg(msgE)
		case <-r.stopChan:
			return
		}
	}
}

func (r *router) OnStart() error {
	go r.poll()
	return nil
}

// InterceptedTransport wraps around MConnTransport
type InterceptedTransport struct {
	service.BaseService
	mConnTransport *MConnTransport
	controller     Controller
	logger         log.Logger
	// Messages going to the controller
	outChan    chan ControllerMsgEnvelop
	router     *router
	routerLock *sync.Mutex
}

// NewInterceptedTransport create InterceptedTransport
func NewInterceptedTransport(
	logger log.Logger,
	mConnConfig conn.MConnConfig,
	channelDescs []*ChannelDescriptor,
	options InterceptedTransportOptions,
	controller Controller,
) *InterceptedTransport {
	t := &InterceptedTransport{
		mConnTransport: NewMConnTransport(
			logger,
			mConnConfig,
			channelDescs,
			options.MConnOptions,
		),
		logger:     logger,
		controller: controller,
		outChan:    controller.SendChan(),
		router:     newRouter(logger, controller.ReceiveChan()),
		routerLock: new(sync.Mutex),
	}
	t.BaseService = *service.NewBaseService(logger.With("transport", "intercepted-transport"), "InterceptedTransport", t)
	return t
}

func (t *InterceptedTransport) Protocols() []Protocol {
	return t.mConnTransport.Protocols()
}

func (t *InterceptedTransport) Endpoints() []Endpoint {
	return t.mConnTransport.Endpoints()
}

func (t *InterceptedTransport) Accept() (Connection, error) {
	c, err := t.mConnTransport.Accept()
	if err != nil {
		return nil, err
	}
	router := t.getRouter()
	return wrapConnection(c, t.outChan, router, t.logger), nil
}

func (t *InterceptedTransport) Dial(ctx context.Context, endpoint Endpoint) (Connection, error) {
	c, err := t.mConnTransport.Dial(ctx, endpoint)
	if err != nil {
		return c, err
	}
	router := t.getRouter()
	return wrapConnection(c, t.outChan, router, t.logger), nil
}

func (t *InterceptedTransport) Listen(endpoint Endpoint) error {
	return t.mConnTransport.Listen(endpoint)
}

func (t *InterceptedTransport) Close() error {
	return t.mConnTransport.Close()
}

func (t *InterceptedTransport) OnStart() error {
	router := t.getRouter()
	err := router.Start()
	if err != nil {
		return err
	}
	return t.controller.Start()
}

func (t *InterceptedTransport) OnStop() {
	router := t.getRouter()
	router.Stop()
	t.controller.Stop()
}

func (t *InterceptedTransport) OnReset() error {
	err := t.controller.Reset()
	if err != nil {
		return err
	}
	router := t.getRouter()
	err = router.Stop()
	if err != nil {
		return err
	}
	router = newRouter(t.logger, t.controller.ReceiveChan())
	t.setRouter(router)
	router.Start()
	return nil
}

func (t *InterceptedTransport) getRouter() *router {
	t.routerLock.Lock()
	defer t.routerLock.Unlock()
	return t.router
}

func (t *InterceptedTransport) setRouter(r *router) {
	t.routerLock.Lock()
	t.router = r
	t.routerLock.Unlock()
}

type InterceptedConnection struct {
	conn    Connection
	inChan  chan ControllerMsgEnvelop
	outChan chan ControllerMsgEnvelop
	router  *router
	peerId  NodeID
	closeCh chan bool
	once    *sync.Once
	logger  log.Logger
}

func wrapConnection(conn Connection, outChan chan ControllerMsgEnvelop, router *router, logger log.Logger) *InterceptedConnection {
	return &InterceptedConnection{
		conn:    conn,
		outChan: outChan,
		inChan:  make(chan ControllerMsgEnvelop, 10),
		router:  router,
		closeCh: make(chan bool, 1),
		once:    new(sync.Once),
		logger:  logger,
	}
}

func (c *InterceptedConnection) Conn() net.Conn {
	return c.conn.Conn()
}

func (c *InterceptedConnection) String() string {
	return c.conn.String()
}

func (c *InterceptedConnection) ReceiveChan() chan ControllerMsgEnvelop {
	return c.inChan
}

func (c *InterceptedConnection) Handshake(
	ctx context.Context,
	nodeInfo NodeInfo,
	privKey crypto.PrivKey,
) (NodeInfo, crypto.PubKey, error) {
	nodeInfo, pubKey, err := c.conn.Handshake(ctx, nodeInfo, privKey)
	if err != nil {
		return nodeInfo, pubKey, err
	}
	c.router.AddConnection(nodeInfo.ID(), c)
	c.peerId = nodeInfo.ID()
	return nodeInfo, pubKey, err
}

func (c *InterceptedConnection) ReceiveMessage() (ChannelID, []byte, error) {
	select {
	case msg := <-c.inChan:
		return msg.ChannelID, msg.MsgB, nil
	case <-c.closeCh:
		return 0, nil, io.EOF
	}
}

func (c *InterceptedConnection) SendMessage(chID ChannelID, msgB []byte) (bool, error) {
	// c.logger.Info("Sending message", "msg", string(msgB))
	select {
	case c.outChan <- ControllerMsgEnvelop{
		ChannelID: chID,
		To:        c.peerId,
		MsgB:      msgB,
	}:
		return true, nil
	case <-c.closeCh:
		return false, io.EOF
	case <-time.After(conn.DefaultSendTimeout):
		return false, nil
	}
}

func (c *InterceptedConnection) TrySendMessage(chID ChannelID, msgB []byte) (bool, error) {
	select {
	case c.outChan <- ControllerMsgEnvelop{
		ChannelID: chID,
		To:        c.peerId,
		MsgB:      msgB,
	}:
		return true, nil
	case <-c.closeCh:
		return false, io.EOF
	default:
		return false, nil
	}
}

func (c *InterceptedConnection) LocalEndpoint() Endpoint {
	return c.conn.LocalEndpoint()
}

func (c *InterceptedConnection) RemoteEndpoint() Endpoint {
	return c.conn.RemoteEndpoint()
}

func (c *InterceptedConnection) Close() error {
	c.once.Do(func() {
		close(c.closeCh)
	})
	return c.conn.Close()
}

func (c *InterceptedConnection) FlushClose() error {
	close(c.closeCh)
	return c.conn.FlushClose()
}

func (c *InterceptedConnection) Status() conn.ConnectionStatus {
	return c.conn.Status()
}
