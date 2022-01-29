package mainpath

import (
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-test/common"
	"github.com/netrixframework/tendermint-test/util"
)

func QuorumPrevotes(sysParams *common.SystemParams) *testlib.TestCase {

	filters := testlib.NewFilterSet()

	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageType(util.Proposal)),
		).Then(
			common.RecordProposal("proposal"),
			testlib.DeliverMessage(),
		),
	)

	filters.AddFilter(
		testlib.If(
			testlib.IsMessageReceive().
				And(common.IsMessageToPart("h")).
				And(common.IsMessageType(util.Prevote)).
				And(common.IsVoteForProposal("proposal")),
		).Then(
			testlib.Count("prevotesDelivered").Incr(),
		),
	)

	sm := testlib.NewStateMachine()
	init := sm.Builder()

	quorumDelivered := init.On(
		testlib.Count("prevotesDelivered").Geq(2*sysParams.F+1),
		"quorumDelivered",
	)
	quorumDelivered.On(
		testlib.IsMessageSend().
			And(common.IsVoteFromPart("h")).
			And(common.IsMessageType(util.Precommit)).
			And(common.IsVoteForProposal("proposal")),
		testlib.SuccessStateLabel,
	)

	testcase := testlib.NewTestCase(
		"QuorumPrevotes",
		1*time.Minute,
		sm,
		filters,
	)
	testcase.SetupFunc(common.Setup(sysParams))
	return testcase
}
