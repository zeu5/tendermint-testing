package node

import (
	"path/filepath"
	"sync"

	cfg "github.com/tendermint/tendermint/config"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
)

const (
	RunningState = "RUNNING"
	StoppedState = "STOPPED"
)

type NodeManager struct {
	config        *cfg.Config
	logger        log.Logger
	NodeInfo      p2p.NodeInfo
	node          *Node
	privVal       *privval.FilePVKey
	nodeLock      *sync.Mutex
	state         string
	stateLock     *sync.Mutex
	transport     p2p.Transport
	transportLock *sync.Mutex
}

func NewNodeManager(config *cfg.Config, logger log.Logger) (*NodeManager, error) {
	m := &NodeManager{
		config:        config,
		logger:        logger,
		state:         StoppedState,
		stateLock:     new(sync.Mutex),
		nodeLock:      new(sync.Mutex),
		transport:     nil,
		transportLock: new(sync.Mutex),
	}
	err := m.createNode()
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (n *NodeManager) createNode() error {
	node, err := DefaultNewNodeWithTransportCreator(n.config, n.logger, n.transportCreator)
	if err != nil {
		return err
	}
	n.logger.Info("Created node")
	n.nodeLock.Lock()
	n.node = node
	n.nodeLock.Unlock()
	return nil
}

func (n *NodeManager) transportCreator(privVal *privval.FilePVKey) transportCreator {
	n.nodeLock.Lock()
	n.privVal = privVal
	n.nodeLock.Unlock()

	return func(logger log.Logger, config *cfg.Config, nodeInfo p2p.NodeInfo, chainID string) p2p.Transport {
		transport := n.Transport()
		if transport != nil {
			return transport
		}
		n.NodeInfo = nodeInfo
		defer n.logger.Info("Created transport")
		if config.P2P.TestIntercept {

			n.nodeLock.Lock()
			privKey := n.privVal
			n.nodeLock.Unlock()

			keyS, err := tmjson.Marshal(privKey)
			if err != nil {
				keyS = []byte{}
			}

			controller := p2p.NewMasterController(
				p2p.MasterControllerConfig{
					MasterAddr: config.P2P.ControllerMasterAddr,
					ListenAddr: config.P2P.ControllerListenAddr,
					ExternAddr: config.P2P.ControllerExternAddr,
					NodeID:     nodeInfo.NodeID,
					NodeInfo: map[string]interface{}{
						"info":     nodeInfo,
						"chain_id": chainID,
						"privkey":  string(keyS),
					},
					BufSize: 10,
				},
				n,
				logger,
			)

			return p2p.NewInterceptedTransport(
				logger,
				p2p.MConnConfig(config.P2P),
				[]*p2p.ChannelDescriptor{},
				p2p.InterceptedTransportOptions{
					MConnOptions: p2p.MConnTransportOptions{
						MaxAcceptedConnections: uint32(config.P2P.MaxNumInboundPeers +
							len(splitAndTrimEmpty(config.P2P.UnconditionalPeerIDs, ",", " ")),
						),
					},
				},
				controller,
			)
		}
		return p2p.NewMConnTransport(
			logger, p2p.MConnConfig(config.P2P), []*p2p.ChannelDescriptor{},
			p2p.MConnTransportOptions{
				MaxAcceptedConnections: uint32(config.P2P.MaxNumInboundPeers +
					len(splitAndTrimEmpty(config.P2P.UnconditionalPeerIDs, ",", " ")),
				),
			},
		)
	}
}

func (n *NodeManager) startTransport() error {
	n.logger.Info("Starting transport")
	transport := n.Transport()

	if transport == nil {
		return nil
	}

	node := n.Node()
	addr, err := p2p.NewNetAddressString(p2p.IDAddressString(node.nodeKey.ID, node.config.P2P.ListenAddress))
	if err != nil {
		return err
	}
	trans, ok := transport.(*p2p.MConnTransport)
	if ok {
		if err := trans.Listen(addr.Endpoint()); err != nil {
			return err
		}
	} else {
		iTrans, ok := transport.(*p2p.InterceptedTransport)
		if ok {
			if err := iTrans.Listen(addr.Endpoint()); err != nil {
				return err
			}
			if err = iTrans.Start(); err != nil {
				return err
			}
		}
	}

	node.isListening = true
	return nil
}

func (n *NodeManager) Transport() p2p.Transport {
	n.transportLock.Lock()
	defer n.transportLock.Unlock()
	return n.transport
}

func (n *NodeManager) Node() *Node {
	n.nodeLock.Lock()
	defer n.nodeLock.Unlock()
	return n.node
}

func (n *NodeManager) State() string {
	n.stateLock.Lock()
	defer n.stateLock.Unlock()
	return n.state
}

func (n *NodeManager) SetStateRunning() {
	n.stateLock.Lock()
	defer n.stateLock.Unlock()
	n.state = RunningState
}

func (n *NodeManager) SetStateStopped() {
	n.stateLock.Lock()
	defer n.stateLock.Unlock()
	n.state = StoppedState
}

func (n *NodeManager) Start() error {
	n.logger.Info("Starting node")
	if n.State() == RunningState {
		return nil
	}
	if n.Transport() == nil {
		n.transportLock.Lock()
		n.transport = n.node.transport
		n.transportLock.Unlock()
		if err := n.startTransport(); err != nil {
			return err
		}
	}
	node := n.Node()
	if err := node.Start(); err != nil {
		return err
	}
	n.SetStateRunning()
	return nil
}

func (n *NodeManager) Stop() error {
	n.logger.Info("Stopping node")
	if n.State() == StoppedState {
		return nil
	}
	node := n.Node()
	err := node.Stop()
	if err != nil {
		return err
	}
	n.SetStateStopped()
	return nil
}

func (n *NodeManager) Restart() error {
	n.logger.Info("Restarting node")
	if err := n.Stop(); err != nil {
		return err
	}
	if err := n.cleanupData(); err != nil {
		return err
	}
	if err := n.createNode(); err != nil {
		return err
	}
	n.Start()
	return nil
}

func (n *NodeManager) Destroy() error {
	n.logger.Info("Destroying node")
	if err := n.Stop(); err != nil {
		return err
	}

	transport := n.Transport()
	if err := transport.Close(); err != nil {
		n.logger.Error("Error closing transport", "err", err)
		return err
	}
	n.node.isListening = false
	return nil
}

func (n *NodeManager) cleanupData() error {
	// Work around for resetting the statefile
	node := n.Node()
	pVal, ok := node.privValidator.(*privval.FilePV)
	if ok {
		pVal.Reset()
	}

	dbPath := n.config.DBDir()
	dirs, err := tmos.ListDir(dbPath)
	if err != nil {
		return err
	}
	for _, d := range dirs {
		if d.IsDir() {
			if err := tmos.RemoveAll(filepath.Join(dbPath, d.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
