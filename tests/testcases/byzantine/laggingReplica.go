package byzantine

import (
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-test/common"
	"github.com/netrixframework/tendermint-test/util"
)

func LaggingReplica(sp *common.SystemParams, rounds int, timeout time.Duration) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	init := sm.Builder()
	init.On(common.IsCommit(), testlib.FailStateLabel)

	allowCatchUp := init.On(common.RoundReached(rounds), "allowCatchUp")
	allowCatchUp.On(
		common.IsCommit(),
		testlib.SuccessStateLabel,
	)
	allowCatchUp.On(
		common.DiffCommits(),
		testlib.FailStateLabel,
	)

	filters := testlib.NewFilterSet()
	filters.AddFilter(common.TrackRoundTwoThirds)
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
			testlib.DropMessage(),
		),
	)
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromRound(0)).
				And(common.IsMessageToPart("h")).
				And(common.IsMessageType(util.Prevote).Or(common.IsMessageType(util.Precommit))).
				And(sm.InState("allowCatchUp").Not()),
		).Then(
			testlib.DropMessage(),
		),
	)

	testcase := testlib.NewTestCase(
		"LaggingReplica",
		timeout,
		sm,
		filters,
	)
	testcase.SetupFunc(common.Setup(sp))
	return testcase
}
