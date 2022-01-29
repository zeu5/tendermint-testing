package rskip

import (
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-test/common"
	"github.com/netrixframework/tendermint-test/util"
)

func RoundSkip(sysParams *common.SystemParams, height, round int) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	roundReached := sm.Builder().
		On(common.HeightReached(height), "SkipRounds").
		On(common.RoundReached(round), "roundReached")

	roundReached.MarkSuccess()
	roundReached.On(
		common.DiffCommits(),
		testlib.FailStateLabel,
	)

	filters := testlib.NewFilterSet()
	filters.AddFilter(common.TrackRoundAll)
	filters.AddFilter(
		testlib.If(
			common.IsFromHeight(height).Not(),
		).Then(
			testlib.DeliverMessage(),
		),
	)
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsFromHeight(height)).
				And(common.IsVoteFromFaulty()),
		).Then(
			common.ChangeVoteToNil(),
		),
	)
	filters.AddFilter(
		testlib.If(
			sm.InState("roundReached"),
		).Then(
			testlib.Set("DelayedPrevotes").DeliverAll(),
		),
	)
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsFromHeight(height)).
				And(common.IsMessageFromPart("h")).
				And(common.IsMessageType(util.Prevote)),
		).Then(
			testlib.Set("DelayedPrevotes").Store(),
			testlib.DropMessage(),
		),
	)

	testCase := testlib.NewTestCase(
		"RoundSkipWithPrevotes",
		30*time.Second,
		sm,
		filters,
	)
	testCase.SetupFunc(common.Setup(sysParams))
	return testCase
}
