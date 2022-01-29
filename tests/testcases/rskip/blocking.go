package rskip

import (
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-test/common"
	"github.com/netrixframework/tendermint-test/util"
)

func BlockVotes(sysParams *common.SystemParams) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	init := sm.Builder()
	init.MarkSuccess()
	init.On(common.IsCommit(), testlib.FailStateLabel)
	init.On(common.IsEventNewRound(1), testlib.FailStateLabel)

	filters := testlib.NewFilterSet()
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageType(util.Prevote)).
				And(testlib.CountTo("votes").Lt(2*sysParams.F)),
		).Then(
			testlib.CountTo("votes").Incr(),
			testlib.DeliverMessage(),
		),
	)
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageType(util.Prevote)).
				And(testlib.CountTo("votes").Geq(2 * sysParams.F)),
		).Then(
			testlib.DropMessage(),
		),
	)

	testcase := testlib.NewTestCase(
		"BlockVotes",
		50*time.Second,
		sm,
		filters,
	)
	testcase.SetupFunc(common.Setup(sysParams))
	return testcase
}
