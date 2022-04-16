package mainpath

import (
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-test/common"
	"github.com/netrixframework/tendermint-test/util"
)

func ProposalNilPrevote(sp *common.SystemParams) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	init := sm.Builder()

	init.On(
		testlib.IsMessageSend().
			And(common.IsMessageFromRound(0)).
			And(common.IsVoteFromPart("h")).
			And(common.IsNotNilVote()),
		testlib.FailStateLabel,
	)
	init.On(
		testlib.IsMessageSend().
			And(common.IsMessageFromRound(0)).
			And(common.IsVoteFromPart("h")).
			And(common.IsNilVote()),
		testlib.SuccessStateLabel,
	)

	cascade := testlib.NewFilterSet()

	cascade.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromRound(0)).
				And(common.IsMessageToPart("h")).
				And(common.IsMessageType(util.Proposal)),
		).Then(
			testlib.DropMessage(),
		),
	)

	testcase := testlib.NewTestCase(
		"ProposalNilPrevote",
		30*time.Second,
		sm,
		cascade,
	)
	testcase.SetupFunc(common.Setup(sp))
	return testcase
}
