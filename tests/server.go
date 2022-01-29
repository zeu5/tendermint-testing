package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/netrixframework/netrix/config"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-test/common"
	"github.com/netrixframework/tendermint-test/testcases/invariant"
	"github.com/netrixframework/tendermint-test/util"
)

func main() {

	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, os.Interrupt, syscall.SIGTERM)

	sysParams := common.NewSystemParams(4)

	server, err := testlib.NewTestingServer(
		&config.Config{
			APIServerAddr: "10.0.0.8:7074",
			NumReplicas:   sysParams.N,
			LogConfig: config.LogConfig{
				Format: "json",
				Path:   "/tmp/tendermint/log/checker.log",
			},
		},
		&util.TMessageParser{},
		[]*testlib.TestCase{
			// testcases.DummyTestCase(),
			// rskip.RoundSkip(sysParams, 1, 2),
			// rskip.BlockVotes(sysParams),
			// rskip.ExpectNewRound(sysParams),
			// rskip.CommitAfterRoundSkip(sysParams),
			// lockedvalue.DifferentDecisions(sysParams),
			// lockedvalue.ExpectUnlock(sysParams),
			// lockedvalue.ExpectNoUnlock(sysParams),
			// lockedvalue.Relocked(sysParams),
			// lockedvalue.LockedCommit(sysParams),
			// mainpath.NilPrevotes(sysParams),
			// mainpath.ProposalNilPrevote(sysParams),
			// mainpath.ProposePrevote(sysParams),
			// mainpath.QuorumPrevotes(sysParams),
			// invariant.QuorumPrecommits(sysParams),
			// invariant.NotNilDecide(sysParams),
			// byzantine.LaggingReplica(sysParams, 10, 10*time.Minute),
			// byzantine.GarbledMessage(sysParams),
			// byzantine.HigherRound(sysParams),
			// byzantine.CrashReplica(sysParams),
			// byzantine.ForeverLaggingReplica(sysParams),
			invariant.PrecommitsInvariant(sysParams),
		},
	)

	if err != nil {
		fmt.Printf("Failed to start server: %s\n", err.Error())
		os.Exit(1)
	}

	go func() {
		<-termCh
		server.Stop()
	}()

	server.Start()

}
