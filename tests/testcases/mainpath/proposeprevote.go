package mainpath

import (
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-test/common"
	"github.com/netrixframework/tendermint-test/util"
)

func ProposePrevote(sp *common.SystemParams) *testlib.TestCase {
	sm := testlib.NewStateMachine()

	init := sm.Builder()
	init.On(
		testlib.IsMessageSend().
			And(common.IsVoteFromPart("h")).
			And(common.IsVoteForProposal("zeroProposal")),
		testlib.SuccessStateLabel,
	)

	filters := testlib.NewFilterSet()
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromRound(0)).
				And(common.IsMessageType(util.Proposal)),
		).Then(
			common.RecordProposal("zeroProposal"),
			testlib.DeliverMessage(),
		),
	)
	testcase := testlib.NewTestCase(
		"ProposePrevote",
		30*time.Second,
		sm,
		filters,
	)
	testcase.SetupFunc(common.Setup(sp))
	return testcase
}
