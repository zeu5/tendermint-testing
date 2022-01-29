package rskip

import (
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-test/common"
	"github.com/netrixframework/tendermint-test/util"
)

func CommitAfterRoundSkip(sp *common.SystemParams) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	init := sm.Builder()

	roundOne := init.On(
		common.RoundReached(1),
		"Round1",
	)
	roundOne.On(
		common.IsCommitForProposal("zeroProposal"),
		testlib.SuccessStateLabel,
	)
	roundOne.On(
		common.DiffCommits(),
		testlib.FailStateLabel,
	)

	filters := testlib.NewFilterSet()
	filters.AddFilter(common.TrackRoundAll)
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsVoteFromFaulty()),
		).Then(
			common.ChangeVoteToNil(),
		),
	)
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromRound(0)).
				And(common.IsVoteFromPart("h")),
		).Then(
			testlib.Set("delayedVotes").Store(),
			testlib.DropMessage(),
		),
	)
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().Not().
				And(sm.InState("Round1")),
		).Then(
			testlib.Set("delayedVotes").DeliverAll(),
			testlib.DeliverMessage(),
		),
	)
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
		"CommitAfterRoundSkip",
		2*time.Minute,
		sm,
		filters,
	)
	testcase.SetupFunc(common.Setup(sp))
	return testcase
}
