package p2p_test

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/p2p/conn"
)

func withController(t *testing.T, tester func(*testing.T, p2p.Controller, *p2p.InterceptedTransport)) {
	controller := p2p.NewDummyController(log.TestingLogger(), 100)

	transport := p2p.NewInterceptedTransport(
		log.TestingLogger(),
		conn.DefaultMConnConfig(),
		[]*p2p.ChannelDescriptor{{ID: byte(chID), Priority: 1}},
		p2p.InterceptedTransportOptions{
			MConnOptions: p2p.MConnTransportOptions{},
		},
		controller,
	)
	err := transport.Listen(p2p.Endpoint{
		Protocol: p2p.MConnProtocol,
		IP:       net.IPv4(127, 0, 0, 1),
		Port:     8080,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, transport.Close())
	})

	tester(t, controller, transport)
}

func init() {
	testTransports["intercept"] = func(t *testing.T) p2p.Transport {
		controller := p2p.NewDummyController(log.TestingLogger(), 100)
		transport := p2p.NewInterceptedTransport(
			log.TestingLogger(),
			conn.DefaultMConnConfig(),
			[]*p2p.ChannelDescriptor{{ID: byte(chID), Priority: 1}},
			p2p.InterceptedTransportOptions{
				MConnOptions: p2p.MConnTransportOptions{},
			},
			controller,
		)
		err := transport.Listen(p2p.Endpoint{
			Protocol: p2p.MConnProtocol,
			IP:       net.IPv4(127, 0, 0, 1),
			Port:     8080,
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, transport.Close())
		})

		return transport
	}
}
