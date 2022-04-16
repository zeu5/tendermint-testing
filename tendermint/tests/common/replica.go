package common

import (
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
)

func RandomReplicaFromPart(partS string) testlib.ReplicaFunc {
	return func(e *types.Event, c *testlib.Context) (types.ReplicaID, bool) {
		partition, ok := getPartition(c)
		if !ok {
			return "", false
		}
		part, ok := partition.GetPart(partS)
		if !ok {
			return "", false
		}
		return part.ReplicaSet.GetRandom().ID, true
	}
}
