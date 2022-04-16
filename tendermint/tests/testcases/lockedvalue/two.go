package lockedvalue

import (
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-test/common"
	"github.com/netrixframework/tendermint-test/util"
)

func ExpectNoUnlock(sysParams *common.SystemParams) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	init := sm.Builder()

	roundOne := init.On(common.RoundReached(1), "RoundOne")

	roundOne.On(
		testlib.IsMessageSend().
			And(common.IsMessageFromRound(1).Not()).
			And(common.IsMessageFromRound(0).Not()).
			And(common.IsVoteFromPart("h")).
			And(common.IsVoteForProposal("zeroProposal")),
		testlib.SuccessStateLabel,
	)
	roundOne.On(
		testlib.IsMessageSend().
			And(common.IsMessageFromRound(1).Not()).
			And(common.IsMessageFromRound(0).Not()).
			And(common.IsVoteFromPart("h")).
			And(common.IsVoteForProposal("zeroProposal").Not()),
		testlib.FailStateLabel,
	)
	init.On(
		common.IsCommit(), testlib.FailStateLabel,
	)

	filters := testlib.NewFilterSet()
	filters.AddFilter(common.TrackRoundAll)
	// Change faulty replicas votes to nil
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsVoteFromFaulty()),
		).Then(common.ChangeVoteToNil()),
	)
	// Record round 0 proposal
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromRound(0)).
				And(common.IsMessageType(util.Proposal)),
		).Then(
			common.RecordProposal("zeroProposal"),
		),
	)
	// Do not deliver votes from "h"
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsVoteFromPart("h")),
		).Then(
			testlib.Set("zeroDelayedPrevotes").Store(),
			testlib.DropMessage(),
		),
	)
	// For higher rounds, we do not deliver proposal until we see a new one
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromRound(0).Not()).
				And(common.IsProposalEq("zeroProposal")),
		).Then(
			testlib.DropMessage(),
		),
	)

	testcase := testlib.NewTestCase(
		"ExpectNoUnlock",
		1*time.Minute,
		sm,
		filters,
	)
	testcase.SetupFunc(common.Setup(sysParams))
	return testcase
}
