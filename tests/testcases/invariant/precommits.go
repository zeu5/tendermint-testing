package invariant

import (
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-test/common"
	"github.com/netrixframework/tendermint-test/util"
)

// When quorum precommit and are delivered, you expect a decision
func QuorumPrecommits(sp *common.SystemParams) *testlib.TestCase {
	filters := testlib.NewFilterSet()
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromRound(0)).
				And(common.IsMessageType(util.Proposal)),
		).Then(
			common.RecordProposal("proposal"),
			testlib.DeliverMessage(),
		),
	)
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageToPart("h")).
				And(common.IsMessageType(util.Precommit)).
				And(common.IsVoteForProposal("proposal")),
		).Then(
			testlib.Count("precommitsSeen").Incr(),
		),
	)

	sm := testlib.NewStateMachine()
	init := sm.Builder()
	init.On(
		common.IsCommitForProposal("proposal"),
		testlib.SuccessStateLabel,
	)
	init.On(
		testlib.Count("precommitsSeen").Geq(2*sp.F+1),
		"quorumPrecommitsSeen",
	).On(
		common.IsCommitForProposal("proposal"),
		testlib.SuccessStateLabel,
	)

	testcase := testlib.NewTestCase(
		"QuorumPrecommits",
		1*time.Minute,
		sm,
		filters,
	)
	testcase.SetupFunc(common.Setup(sp))
	return testcase
}
