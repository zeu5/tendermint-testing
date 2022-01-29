package rskip

import (
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-test/common"
	"github.com/netrixframework/tendermint-test/util"
)

func ExpectNewRound(sp *common.SystemParams) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	init := sm.Builder()
	// We want replicas in partition "h" to move to round 1
	init.On(
		common.IsNewHeightRoundFromPart("h", 1, 1),
		testlib.SuccessStateLabel,
	)
	newRound := init.On(
		testlib.Count("round1ToH").Geq(sp.F+1),
		"newRoundMessagesDelivered",
	).On(
		common.IsNewHeightRoundFromPart("h", 1, 1),
		"NewRound",
	)
	newRound.On(
		common.DiffCommits(),
		testlib.FailStateLabel,
	)
	newRound.On(
		common.IsCommit(),
		testlib.SuccessStateLabel,
	)

	init.On(
		common.IsCommit(),
		testlib.FailStateLabel,
	)

	filters := testlib.NewFilterSet()
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
			testlib.IsMessageReceive().
				And(common.IsMessageFromRound(1)).
				And(common.IsMessageToPart("h")).
				And(
					common.IsMessageType(util.Proposal).
						Or(common.IsMessageType(util.Prevote)).
						Or(common.IsMessageType(util.Precommit)),
				),
		).Then(
			testlib.Count("round1ToH").Incr(),
		),
	)
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromRound(0)).
				And(common.IsMessageToPart("h")).
				And(common.IsMessageType(util.Prevote).Or(common.IsMessageType(util.Precommit))),
		).Then(
			testlib.DropMessage(),
		),
	)
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromRound(0)).
				And(common.IsVoteFromPart("h")),
		).Then(
			testlib.DropMessage(),
		),
	)

	testcase := testlib.NewTestCase(
		"ExpectNewRound",
		1*time.Minute,
		sm,
		filters,
	)
	testcase.SetupFunc(common.Setup(sp))
	return testcase
}
