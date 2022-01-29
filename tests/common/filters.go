package common

import (
	"fmt"
	"strconv"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
)

func TrackRoundAll(e *types.Event, c *testlib.Context) (messages []*types.Message, handled bool) {
	eType, ok := e.Type.(*types.GenericEventType)
	if !ok {
		return
	}
	if eType.T != "newStep" {
		return
	}
	roundS, ok := eType.Params["round"]
	if !ok {
		return
	}
	round, err := strconv.Atoi(roundS)
	if err != nil {
		return
	}
	roundKey := fmt.Sprintf("_roundCount_%d", round)
	replicaRoundKey := fmt.Sprintf("_roundCount_%s_%d", e.Replica, round)

	if c.Vars.Exists(replicaRoundKey) {
		return
	}
	c.Vars.Set(replicaRoundKey, true)
	roundCount, ok := c.Vars.GetCounter(roundKey)
	n, _ := c.Vars.GetInt("n")
	if !ok {
		c.Vars.SetCounter(roundKey)
		roundCount, _ = c.Vars.GetCounter(roundKey)
	}
	roundCount.Incr()
	if roundCount.Value() == n {
		curRound, ok := c.Vars.GetInt(curRoundKey)
		if !ok || curRound < round {
			c.Vars.Set(curRoundKey, round)
		}
	}
	return
}

func TrackRoundTwoThirds(e *types.Event, c *testlib.Context) (messages []*types.Message, handled bool) {
	eType, ok := e.Type.(*types.GenericEventType)
	if !ok {
		return
	}
	if eType.T != "newStep" {
		return
	}
	roundS, ok := eType.Params["round"]
	if !ok {
		return
	}
	round, err := strconv.Atoi(roundS)
	if err != nil {
		return
	}
	roundKey := fmt.Sprintf("_roundCount_%d", round)
	replicaRoundKey := fmt.Sprintf("_roundCount_%s_%d", e.Replica, round)

	if c.Vars.Exists(replicaRoundKey) {
		return
	}
	c.Vars.Set(replicaRoundKey, true)
	roundCount, ok := c.Vars.GetCounter(roundKey)
	f, _ := c.Vars.GetInt("f")
	if !ok {
		c.Vars.SetCounter(roundKey)
		roundCount, _ = c.Vars.GetCounter(roundKey)
	}
	roundCount.Incr()
	if roundCount.Value() >= 2*f+1 {
		curRound, ok := c.Vars.GetInt(curRoundKey)
		if !ok || curRound < round {
			c.Vars.Set(curRoundKey, round)
		}
	}
	return
}
