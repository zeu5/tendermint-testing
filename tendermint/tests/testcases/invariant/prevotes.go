package invariant

import (
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-test/common"
	"github.com/netrixframework/tendermint-test/util"
)

func PrecommitsInvariant(sp *common.SystemParams) *testlib.TestCase {
	filters := testlib.NewFilterSet()
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromRound(0)).
				And(common.IsMessageType(util.Proposal)),
		).Then(
			common.RecordProposal("zeroProposal"),
			testlib.DropMessage(),
		),
	)

	sm := testlib.NewStateMachine()
	init := sm.Builder()
	init.On(
		testlib.IsMessageSend().
			And(common.IsMessageFromRound(0)).
			And(common.IsMessageType(util.Precommit)).
			And(common.IsVoteForProposal("zeroProposal")),
		testlib.FailStateLabel,
	)
	init.MarkSuccess()

	testcase := testlib.NewTestCase(
		"PrecommitInvariant",
		1*time.Minute,
		sm,
		filters,
	)
	return testcase
}
