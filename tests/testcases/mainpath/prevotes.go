package mainpath

import (
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-test/common"
	"github.com/netrixframework/tendermint-test/util"
)

func NilPrevotes(sysParams *common.SystemParams) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	init := sm.Builder()

	nilQuorumDelivered := init.On(
		testlib.Count("nilPrevotesDelivered").Geq(2*sysParams.F+1),
		"nilQuorumDelivered",
	)
	nilQuorumDelivered.On(
		testlib.IsMessageSend().
			And(testlib.IsMessageFromF(common.GetRandomReplica)).
			And(common.IsMessageType(util.Precommit)).
			And(common.IsNilVote()),
		testlib.SuccessStateLabel,
	)

	filters := testlib.NewFilterSet()
	// We don't deliver any proposal and hence we should see that replicas other than the proposer prevote nil.
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageType(util.Proposal)),
		).Then(
			testlib.DropMessage(),
		),
	)
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageReceive().
				And(testlib.IsMessageToF(common.GetRandomReplica)).
				And(common.IsMessageType(util.Prevote)).
				And(common.IsNilVote()),
		).Then(
			testlib.Count("nilPrevotesDelivered").Incr(),
		),
	)

	testcase := testlib.NewTestCase(
		"NilPrevotes",
		1*time.Minute,
		sm,
		filters,
	)
	testcase.SetupFunc(common.Setup(sysParams, common.PickRandomReplica()))
	return testcase
}
