package invariant

import (
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-test/common"
)

func NotNilDecide(sp *common.SystemParams) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	init := sm.Builder()
	init.MarkSuccess()
	init.On(
		common.IsNilCommit(),
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
			testlib.IsMessageSend().
				And(common.IsVoteFromPart("h")),
		).Then(
			testlib.DropMessage(),
		),
	)

	testcase := testlib.NewTestCase(
		"NotNilDecide",
		2*time.Minute,
		sm,
		filters,
	)
	testcase.SetupFunc(common.Setup(sp))
	return testcase
}
