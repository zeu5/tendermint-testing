package byzantine

import (
	"math/rand"
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-test/common"
	"github.com/netrixframework/tendermint-test/util"
)

func GarbledMessage(sysParams *common.SystemParams) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	sm.Builder().On(
		common.IsCommit(),
		testlib.SuccessStateLabel,
	)

	filters := testlib.NewFilterSet()
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromPart("faulty")),
		).Then(
			garbleMessage(),
		),
	)

	testcase := testlib.NewTestCase(
		"GarbledMessages",
		2*time.Minute,
		sm,
		filters,
	)
	testcase.SetupFunc(common.Setup(sysParams))
	return testcase
}

func garbleMessage() testlib.Action {
	return func(e *types.Event, c *testlib.Context) []*types.Message {
		m, ok := c.GetMessage(e)
		if !ok {
			return []*types.Message{}
		}
		tMsg, ok := util.GetParsedMessage(m)
		if !ok {
			return []*types.Message{m}
		}
		randBytes := make([]byte, 100)
		rand.Read(randBytes)
		tMsg.MsgB = randBytes
		tMsg.Data = nil
		newMsg, err := tMsg.Marshal()
		if err != nil {
			return []*types.Message{m}
		}
		return []*types.Message{c.NewMessage(m, newMsg)}
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
